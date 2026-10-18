//go:build !windows

package components

import "syscall"

func genericPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err != syscall.ESRCH
}
