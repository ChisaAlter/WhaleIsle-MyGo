//go:build windows

package mygo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

var (
	siKernel32            = syscall.NewLazyDLL("kernel32.dll")
	siCreateMutexW        = siKernel32.NewProc("CreateMutexW")
	siWaitForSingleObject = siKernel32.NewProc("WaitForSingleObject")
	siReleaseMutex        = siKernel32.NewProc("ReleaseMutex")
	siCloseHandle         = siKernel32.NewProc("CloseHandle")
	siGetShortPathNameW   = siKernel32.NewProc("GetShortPathNameW")
)

const (
	siErrAlreadyExists = syscall.Errno(183)
	siWaitObject0      = 0x00000000
	siWaitAbandoned    = 0x00000080
	siWSAEAddrInUse    = syscall.Errno(10048)
)

type singleInstancePlatform struct {
	mutex uintptr
}

func (p *singleInstancePlatform) release() {
	if p.mutex != 0 {
		siReleaseMutex.Call(p.mutex)
		siCloseHandle.Call(p.mutex)
		p.mutex = 0
	}
}

func singleInstanceMutexName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "Local\\mygo-" + hex.EncodeToString(sum[:12])
}

func acquireSingleInstance(key, path string, msg []byte, appName string) (singleInstancePlatform, bool) {
	mutexName, err := syscall.UTF16PtrFromString(singleInstanceMutexName(key))
	if err != nil {
		return acquireSingleInstanceFallback(path, msg, appName)
	}
	h, _, lastErr := siCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(mutexName)))
	if h == 0 {
		return acquireSingleInstanceFallback(path, msg, appName)
	}
	alreadyExists := errors.Is(lastErr, siErrAlreadyExists)

	if alreadyExists {
		// Another live instance holds the mutex. Wait for its listener to come
		// up (in case both instances launched simultaneously) and hand over.
		deadline := time.Now().Add(3 * time.Second)
		backoff := 15 * time.Millisecond
		for {
			if conn, dialErr := net.DialTimeout("unix", path, time.Second); dialErr == nil {
				if handOverSecondInstance(conn, msg, secondInstanceHandshakeTimeout) {
					if launchedByDev() {
						log.Printf("mygo: %s is already running: handed the command line over to it (RequestSingleInstanceLock)", appName)
					}
					siCloseHandle.Call(h)
					return singleInstancePlatform{}, false
				}
			}
			// Check whether the previous owner exited or crashed while we waited.
			if w, _, _ := siWaitForSingleObject.Call(h, 0); w == siWaitObject0 || w == siWaitAbandoned {
				alreadyExists = false
				break
			}
			if time.Now().After(deadline) {
				// Primary holds the mutex but never accepted a handover; fail
				// open rather than exiting silently without showing a window.
				siCloseHandle.Call(h)
				log.Printf("mygo: %s starts without the single-instance socket: primary did not acknowledge handover", appName)
				return singleInstancePlatform{}, true
			}
			time.Sleep(backoff)
			if backoff < 100*time.Millisecond {
				backoff = min(backoff*2, 100*time.Millisecond)
			}
		}
	}

	// We atomically own the named kernel mutex. Bind the forwarding socket.
	var bindErr error
	for attempt := 0; attempt < 5; attempt++ {
		_ = os.Remove(path)
		ln, err := net.Listen("unix", path)
		if err == nil {
			startSingleInstanceListener(ln, path)
			return singleInstancePlatform{mutex: h}, true
		}
		if !errors.Is(err, syscall.EADDRINUSE) && !errors.Is(err, siWSAEAddrInUse) {
			return singleInstancePlatform{mutex: h}, true
		}
		bindErr = err
		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}
	log.Printf("mygo: %s starts without the single-instance socket: %v", appName, bindErr)
	return singleInstancePlatform{mutex: h}, true
}

func acquireSingleInstanceFallback(path string, msg []byte, appName string) (singleInstancePlatform, bool) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
			if handOverSecondInstance(conn, msg, secondInstanceHandshakeTimeout) {
				if launchedByDev() {
					log.Printf("mygo: %s is already running: handed the command line over to it (RequestSingleInstanceLock)", appName)
				}
				return singleInstancePlatform{}, false
			}
			log.Printf("mygo: %s: the single-instance socket has no live owner; taking it over", appName)
		}
		_ = os.Remove(path)
		ln, err := net.Listen("unix", path)
		if err == nil {
			startSingleInstanceListener(ln, path)
			return singleInstancePlatform{}, true
		}
		if !errors.Is(err, syscall.EADDRINUSE) && !errors.Is(err, siWSAEAddrInUse) {
			return singleInstancePlatform{}, true
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}
	log.Printf("mygo: %s starts without the single-instance socket: %v", appName, lastErr)
	return singleInstancePlatform{}, true
}

func shortPathName(path string) string {
	p16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	buf := make([]uint16, 260)
	n, _, _ := siGetShortPathNameW.Call(uintptr(unsafe.Pointer(p16)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	if int(n) >= len(buf) {
		buf = make([]uint16, n+1)
		n, _, _ = siGetShortPathNameW.Call(uintptr(unsafe.Pointer(p16)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 || int(n) >= len(buf) {
			return ""
		}
	}
	return syscall.UTF16ToString(buf[:n])
}

func dirExists(dir string) bool {
	if dir == "" {
		return false
	}
	fi, err := os.Stat(dir)
	return err == nil && fi.IsDir()
}

// singleInstanceSocket returns a short per-user AF_UNIX socket path on Windows.
// Windows AF_UNIX sun_path holds up to 108 bytes (107 + NUL); when TEMP is
// long, GetShortPathNameW is tried before falling back to shorter Windows temp
// locations (never POSIX "/tmp").
func singleInstanceSocket(name string) string {
	sum := sha256.Sum256([]byte(name))
	file := "mygo-" + strconv.Itoa(os.Getuid()) + "-" + hex.EncodeToString(sum[:6]) + ".sock"
	tryDir := func(dir string, limit int) (string, bool) {
		if dir == "" {
			return "", false
		}
		p := filepath.Join(dir, file)
		if len(p) < limit {
			return p, true
		}
		if short := shortPathName(dir); short != "" {
			sp := filepath.Join(short, file)
			if len(sp) < limit {
				return sp, true
			}
		}
		return "", false
	}
	if p, ok := tryDir(os.Getenv("XDG_RUNTIME_DIR"), 100); ok {
		return p
	}
	tempDir := os.TempDir()
	if p, ok := tryDir(tempDir, 100); ok {
		return p
	}
	if p, ok := tryDir(tempDir, 107); ok {
		return p
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidate := filepath.Join(localAppData, "Temp")
		if dirExists(candidate) {
			if p, ok := tryDir(candidate, 107); ok {
				return p
			}
		}
	}
	if sysRoot := os.Getenv("SystemRoot"); sysRoot != "" {
		candidate := filepath.Join(sysRoot, "Temp")
		if dirExists(candidate) {
			if p, ok := tryDir(candidate, 107); ok {
				return p
			}
		}
	}
	return filepath.Join(tempDir, file)
}
