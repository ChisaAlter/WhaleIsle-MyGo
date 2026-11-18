//go:build windows

package paths

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

// kernel32 GetDriveTypeW is the documented way to classify a volume without
// touching it: DRIVE_REMOTE is 4, and the call does not open the path, so a
// dropped network drive cannot hang it.
var getDriveType = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDriveTypeW")

const driveRemote = 4

// volumeIsRemote reports whether path is on a network volume. A profile on a
// network share makes every session write a round trip and can disappear
// mid-run, so the location chooser refuses it.
func volumeIsRemote(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	root := filepath.VolumeName(abs)
	if root == "" {
		return false
	}
	if root[len(root)-1] != '\\' {
		root += `\`
	}
	target, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return false
	}
	kind, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(target)))
	return kind == driveRemote
}
