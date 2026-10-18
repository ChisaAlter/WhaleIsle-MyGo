//go:build windows

package components

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// Query/synchronize only. This observer never requests process termination rights.
func bridgeAlertProcessIdentity(pid int, exe string) (string, error) {
	h, err := syscall.OpenProcess(0x1000|0x100000, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(h)
	status, err := syscall.WaitForSingleObject(h, 0)
	if err != nil || status != syscall.WAIT_TIMEOUT {
		return "", fmt.Errorf("process not running")
	}
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	ok, _, _ := bridgeQueryImage.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	expected, err := filepath.EvalSymlinks(exe)
	if err == nil {
		expected, err = filepath.Abs(expected)
	}
	if ok == 0 || err != nil || !strings.EqualFold(filepath.Clean(syscall.UTF16ToString(buf[:size])), filepath.Clean(expected)) {
		return "", fmt.Errorf("process image mismatch")
	}
	var created, exited, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &created, &exited, &kernel, &user); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d:%d", pid, created.HighDateTime, created.LowDateTime), nil
}
