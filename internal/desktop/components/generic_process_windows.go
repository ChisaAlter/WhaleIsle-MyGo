//go:build windows

package components

import "syscall"

func genericPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(0x1000|0x100000, false, uint32(pid))
	if err != nil {
		return err != syscall.Errno(87)
	} // Access denied is not evidence of exit.
	defer syscall.CloseHandle(handle)
	result, err := syscall.WaitForSingleObject(handle, 0)
	return err != nil || result == syscall.WAIT_TIMEOUT
}
