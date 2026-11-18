//go:build !windows

package credentials

import (
	"errors"
	"syscall"
)

func lockCreationContended(error) bool { return false }

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
