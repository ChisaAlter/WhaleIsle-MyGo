//go:build windows

package market

import (
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpace = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// platformFreeBytes reports the free bytes available to the caller on dir's
// volume (GetDiskFreeSpaceExW, like the native preflight's fs.statfs).
func platformFreeBytes(dir string) (int64, error) {
	p, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var freeToCaller, total, totalFree uint64
	r1, _, callErr := procGetDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeToCaller)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r1 == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return 0, callErr
		}
		return 0, ErrSpaceUnsupported
	}
	return int64(freeToCaller), nil
}
