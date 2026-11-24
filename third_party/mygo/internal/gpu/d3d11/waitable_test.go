//go:build windows

package d3d11

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

func TestWaitableSwapChainModesAndResize(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cases := []struct {
		name string
		make func(hwnd uintptr) (*Renderer, error)
	}{
		{"hwnd", New},
		{"composed", NewComposed},
		{"composed-rounded", func(hwnd uintptr) (*Renderer, error) { return NewComposedRounded(hwnd, 12) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hwnd := hiddenWindow(t, 240, 160)
			r, err := tc.make(hwnd)
			if err != nil {
				t.Skip("no Direct3D 11:", err)
			}
			defer r.Release()

			// Before the first frame creates the swap chain, WaitHandle is 0 and
			// TryAcquireFrame never blocks the initial render.
			if r.WaitHandle() != 0 {
				t.Fatalf("expected 0 wait handle before first draw, got %#x", r.WaitHandle())
			}
			if !r.TryAcquireFrame() {
				t.Fatal("TryAcquireFrame must succeed before swap chain exists")
			}

			s1 := &scene.Scene{Width: 240, Height: 160, Clear: scene.Color{R: 20, G: 40, B: 60, A: 255}}
			if err := r.Render(s1); err != nil {
				t.Fatalf("first render failed: %v", err)
			}
			if r.swapChain2 == 0 {
				t.Skip("no IDXGISwapChain2 on this system")
			}
			if r.scFlags&dxgiSwapChainFlagWaitable == 0 || r.WaitHandle() == 0 {
				t.Fatalf("expected waitable swap chain handle, flags=%#x handle=%#x", r.scFlags, r.WaitHandle())
			}

			// MarkFrameReady must mark the next slot acquired without blocking.
			r.MarkFrameReady()
			if !r.TryAcquireFrame() {
				t.Fatal("TryAcquireFrame should return true after MarkFrameReady")
			}

			// Resizing must preserve scFlags across ResizeBuffers without DXGI errors.
			s2 := &scene.Scene{Width: 320, Height: 220, Clear: scene.Color{R: 10, G: 20, B: 30, A: 255}}
			if err := r.Render(s2); err != nil {
				t.Fatalf("resized render failed: %v", err)
			}
			if r.slotAcquired {
				t.Fatal("slotAcquired should be cleared after Present")
			}
			if r.WaitHandle() == 0 {
				t.Fatal("waitable handle lost after resize")
			}

			r.Release()
			if r.WaitHandle() != 0 {
				t.Fatalf("waitable handle not cleared after Release: %#x", r.WaitHandle())
			}
		})
	}
}
