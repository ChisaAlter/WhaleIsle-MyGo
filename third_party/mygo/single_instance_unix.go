//go:build unix

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
)

type singleInstancePlatform struct{}

func (p *singleInstancePlatform) release() {}

func acquireSingleInstance(_, path string, msg []byte, appName string) (singleInstancePlatform, bool) {
	lockFile, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err == nil {
		defer lockFile.Close()
		for {
			if err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); !errors.Is(err, syscall.EINTR) {
				break
			}
		}
		if err == nil {
			defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		}
	}

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
			if handOverSecondInstance(conn, msg, secondInstanceHandshakeTimeout) {
				if launchedByDev() {
					// mygo dev would only see the app exit.
					log.Printf("mygo: %s is already running: handed the command line over to it (RequestSingleInstanceLock)", appName)
				}
				return singleInstancePlatform{}, false
			}
			// The socket answered but no owner acknowledged: a crash left it
			// behind, or a surviving child inherited the listening handle. Fall
			// through and take the lock over, so the launch still shows a window.
			log.Printf("mygo: %s: the single-instance socket has no live owner; taking it over", appName)
		}
		// Nobody answers: remove a stale socket left by a crash and take over.
		// Holding flock across Dial+Remove+Listen serializes concurrent
		// launches so a second instance cannot unlink our freshly bound socket.
		_ = os.Remove(path)
		ln, err := net.Listen("unix", path)
		if err == nil {
			startSingleInstanceListener(ln, path)
			return singleInstancePlatform{}, true
		}
		if !errors.Is(err, syscall.EADDRINUSE) {
			// Locking is not possible here (e.g. read-only temp dir):
			// behave as if this were the only instance.
			return singleInstancePlatform{}, true
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}
	// A stale listener that a surviving process still holds keeps the path
	// bound. The app must still start: forwarding is unavailable until that
	// process is gone, but a silent exit with no window is worse.
	log.Printf("mygo: %s starts without the single-instance socket: %v", appName, lastErr)
	return singleInstancePlatform{}, true
}

// singleInstanceSocket returns a short per-user socket path; Unix socket
// paths are limited to about 100 bytes.
func singleInstanceSocket(name string) string {
	sum := sha256.Sum256([]byte(name))
	file := "mygo-" + strconv.Itoa(os.Getuid()) + "-" + hex.EncodeToString(sum[:6]) + ".sock"
	for _, dir := range []string{os.Getenv("XDG_RUNTIME_DIR"), os.TempDir(), "/tmp"} {
		if dir != "" && len(filepath.Join(dir, file)) < 100 {
			return filepath.Join(dir, file)
		}
	}
	return filepath.Join("/tmp", file)
}
