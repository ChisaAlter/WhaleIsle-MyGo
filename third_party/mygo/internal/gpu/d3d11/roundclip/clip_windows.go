//go:build windows && (amd64 || arm64)

// Package roundclip owns an anti-aliased DirectComposition rectangle clip.
// Both bitmap and swap-chain presentation use the same geometry and COM ABI.
package roundclip

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Clip is owned by one composition visual on its render thread.
type Clip struct {
	object                uintptr
	width, height, radius float32
}

func pointer(a uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&a)) }

//go:uintptrescapes
func method(object uintptr, slot int, args ...uintptr) uintptr {
	vt := *(*uintptr)(pointer(object))
	fn := *(*uintptr)(unsafe.Add(pointer(vt), slot*int(unsafe.Sizeof(uintptr(0)))))
	argv := append([]uintptr{object}, args...)
	r, _, _ := syscall.SyscallN(fn, argv...)
	return r
}

func check(operation string, hr uintptr) error {
	if int32(uint32(hr)) < 0 {
		return fmt.Errorf("%s: HRESULT 0x%08X", operation, uint32(hr))
	}
	return nil
}

// Apply sets a device-pixel clip and reports whether it changed, before its
// caller commits the device. SDK
// dcomp.h defines the overload groups; MSVC lays the overloads in reverse
// declaration order: Visual.SetClip(Clip*)=13 and the float property setters
// of RectangleClip=4,6,...,26 (the odd slots take Animation*).
func (c *Clip) Apply(device, visual uintptr, width, height, radius float32) (bool, error) {
	if c.object == 0 {
		if err := check("IDCompositionDevice::CreateRectangleClip", method(device, 24, uintptr(unsafe.Pointer(&c.object)))); err != nil {
			return false, err
		}
		if err := check("IDCompositionVisual::SetClip", method(visual, 13, c.object)); err != nil {
			c.Close()
			return false, err
		}
		c.width, c.height, c.radius = -1, -1, -1
	}
	radius = min(max(radius, 0), min(width, height)/2)
	if c.width == width && c.height == height && c.radius == radius {
		return false, nil
	}
	values := [12]float32{0, 0, width, height, radius, radius, radius, radius, radius, radius, radius, radius}
	for i, value := range values {
		vt := *(*uintptr)(pointer(c.object))
		fn := *(*uintptr)(unsafe.Add(pointer(vt), (4+2*i)*int(unsafe.Sizeof(uintptr(0)))))
		if err := check("IDCompositionRectangleClip property", setFloat(fn, c.object, value)); err != nil {
			return false, err
		}
	}
	c.width, c.height, c.radius = width, height, radius
	return true, nil
}

// Close releases the clip before its visual and device are released.
func (c *Clip) Close() {
	if c.object != 0 {
		method(c.object, 2)
		*c = Clip{}
	}
}
