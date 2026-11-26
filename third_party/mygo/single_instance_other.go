//go:build !windows && !unix

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
		if !errors.Is(err, syscall.EADDRINUSE) {
			return singleInstancePlatform{}, true
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}
	log.Printf("mygo: %s starts without the single-instance socket: %v", appName, lastErr)
	return singleInstancePlatform{}, true
}

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
