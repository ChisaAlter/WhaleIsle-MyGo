//go:build windows

package roundclip

import (
	"math"
	"syscall"
)

// Windows x64 SyscallN mirrors the second argument's bits into XMM1.
func setFloat(fn, object uintptr, value float32) uintptr {
	r, _, _ := syscall.SyscallN(fn, object, uintptr(math.Float32bits(value)))
	return r
}
