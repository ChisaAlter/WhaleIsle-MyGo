//go:build windows

package roundclip

import (
	"math"
	"syscall"
)

func floatThunkAddress() uintptr

// SyscallN performs the normal Go/Windows foreign-call transition. The static
// thunk only transfers the float bits to S0, then tail-calls the COM method.
func setFloat(fn, object uintptr, value float32) uintptr {
	r, _, _ := syscall.SyscallN(floatThunkAddress(), object, uintptr(math.Float32bits(value)), fn)
	return r
}
