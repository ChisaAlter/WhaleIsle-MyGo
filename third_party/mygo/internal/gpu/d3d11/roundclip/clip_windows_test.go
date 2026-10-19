//go:build windows && (amd64 || arm64)

package roundclip

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu/d3d11/device"
)

// A null DXGI rendering device supports a composition property tree without a
// window, target, swap chain or GPU. This exercises real COM float setters on
// the executing architecture; cross-compiling it alone is only build evidence.
func TestClipNativePropertyABIWithoutSurface(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	create := syscall.NewLazyDLL(device.SystemDir() + `\dcomp.dll`).NewProc("DCompositionCreateDevice")
	if err := create.Find(); err != nil {
		t.Fatal(err)
	}
	iid := [16]byte{0x3a, 0xa9, 0x7e, 0xc3, 0xaa, 0xe7, 0x0d, 0x45, 0xb1, 0x6f, 0x97, 0x46, 0xcb, 0x04, 0x07, 0xf3}
	var dev, visual uintptr
	hr, _, _ := create.Call(0, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&dev)))
	if err := check("DCompositionCreateDevice(null)", hr); err != nil {
		t.Fatal(err)
	}
	if dev == 0 {
		t.Fatal("null composition device")
	}
	defer method(dev, 2)
	if err := check("CreateVisual", method(dev, 7, uintptr(unsafe.Pointer(&visual)))); err != nil {
		t.Fatal(err)
	}
	defer method(visual, 2)
	var clip Clip
	defer clip.Close()
	for _, tc := range []struct {
		w, h, r float32
		changed bool
	}{{640, 480, 20, true}, {640, 480, 20, false}, {960, 720, 30, true}, {10, 20, 20, true}, {10, 20, 5, false}, {960, 720, 0, true}} {
		changed, err := clip.Apply(dev, visual, tc.w, tc.h, tc.r)
		if err != nil || changed != tc.changed {
			t.Fatalf("Apply(%v): changed=%v error=%v", tc, changed, err)
		}
		if err := check("Commit property tree", method(dev, 3)); err != nil {
			t.Fatal(err)
		}
	}
	clip.Close()
	clip.Close()
	if clip.object != 0 {
		t.Fatal("clip retained a released COM object")
	}
}
