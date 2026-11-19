//go:build !windows

package harness

import (
	"fmt"
	"os"
	"syscall"
)

// processImage names the executable behind pid through /proc. A pid that is
// gone reads as empty; a platform without /proc reports that it cannot tell, so
// a caller never guesses an identity from a bare pid.
func processImage(pid int) (string, bool) {
	if _, err := os.Stat("/proc/self"); err != nil {
		return "", false
	}
	if pid <= 0 {
		return "", true
	}
	target, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return "", true
	}
	return target, true
}

// reapHostProcess ends the Host's process group; the child was detached into
// its own group at spawn, so its pid is the group's id.
func reapHostProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGKILL)
}
