//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

// TestCompositionProbe answers two questions about the DirectComposition
// device a cold start spends 238-282ms on, and nothing else:
//
//	cd third_party/mygo
//	MYGO_COMP_PROBE=1 go test -count=1 -run TestCompositionProbe -v ./internal/windows
//
//  1. Is that time the work of the call, or the price of the first call in a
//     process? It makes the device several times here, the first cold and the
//     rest after a pause, which shows both the warm price and the one-time one.
//  2. Can the device be made on a worker thread and used from another? A
//     creation that could run beside the boot would have to be, so the probe
//     makes one there and has this thread check it, commit it, and give it a
//     composition target for a window.
func TestCompositionProbe(t *testing.T) {
	if os.Getenv("MYGO_COMP_PROBE") == "" {
		t.Skip("composition probe not requested: set MYGO_COMP_PROBE=1")
	}

	for i := 0; i < 3; i++ {
		start := time.Now()
		c, err := newComposition()
		took := time.Since(start)
		if err != nil {
			t.Fatalf("creation %d: %v", i, err)
		}
		t.Logf("creation %d on this thread: %.1fms", i, float64(took.Microseconds())/1000)
		if i < 2 {
			c.free()
			time.Sleep(1500 * time.Millisecond)
		}
	}

	type made struct {
		c    *composition
		took time.Duration
		err  error
	}
	done := make(chan made, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		hr, _, _ := procCoInitializeEx.Call(0, 0x0) // COINIT_MULTITHREADED
		if failed(hr) {
			done <- made{err: hresultError("CoInitializeEx", hr)}
			return
		}
		start := time.Now()
		c, err := newComposition()
		done <- made{c, time.Since(start), err}
	}()
	m := <-done
	if m.err != nil {
		t.Fatalf("worker creation: %v", m.err)
	}
	t.Logf("creation on a worker thread: %.1fms", float64(m.took.Microseconds())/1000)

	// Does the object made in the worker's apartment work from this one?
	var valid int32
	hr := comCall(m.c.dcomp, dcompDevCheckDeviceState, uintptr(unsafe.Pointer(&valid)))
	t.Logf("CheckDeviceState from this thread: hr=%#08x valid=%d", uint32(hr), valid)

	// And does a draw-side call, the one the renderer makes, work here?
	hr = comCall(m.c.dcomp, dcompDevCommit)
	t.Logf("Commit from this thread: hr=%#08x", uint32(hr))

	// The call the window makes at creation: a composition target for a window
	// the worker never saw. This is the one that would have to cross threads
	// for the device to be made early and used later.
	if err := registerClasses(); err != nil {
		t.Fatalf("registering classes: %v", err)
	}
	hwnd := createWindow(0, windowClass, "composition probe", 0x00CF0000, 0, 0, 200, 200, 0)
	if hwnd == 0 {
		t.Fatalf("no window for the target probe")
	}
	var target uintptr
	hr = comCall(m.c.dcomp, dcompDevCreateTargetForHwnd, hwnd, 1, uintptr(unsafe.Pointer(&target)))
	t.Logf("CreateTargetForHwnd from this thread: hr=%#08x", uint32(hr))
	if target != 0 {
		t.Logf("target made: %#x", target)
	}
	procDestroyWindow.Call(hwnd)
	m.c.free()

	// How the cost recovers with idle time: the difference between a creation
	// right after another and one after a pause is the GPU leaving and
	// returning to a low-power state, not work that could be done in parallel.
	for _, idle := range []time.Duration{0, 100 * time.Millisecond, 300 * time.Millisecond, 1000 * time.Millisecond, 3000 * time.Millisecond} {
		time.Sleep(idle)
		start := time.Now()
		c, err := newComposition()
		took := time.Since(start)
		if err != nil {
			t.Fatalf("after %v idle: %v", idle, err)
		}
		t.Logf("after %5v idle: %.1fms", idle, float64(took.Microseconds())/1000)
		c.free()
	}
}
