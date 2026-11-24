//go:build windows

package d3d11

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

func TestFadeHorizontalGPUMatchesCPU(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	s := &scene.Scene{
		Width:  120,
		Height: 40,
		Clear:  scene.Color{R: 18, G: 24, B: 32, A: 255},
		Ops: []scene.Op{
			// Checkerboard background under the faded region to verify true
			// alpha modulation (not an opaque solid-color overlay).
			{Kind: scene.OpFill, Rect: scene.Rect{X: 10, Y: 4, W: 50, H: 32}, Color: scene.Color{R: 40, G: 120, B: 180, A: 255}},
			{Kind: scene.OpFill, Rect: scene.Rect{X: 60, Y: 4, W: 50, H: 32}, Color: scene.Color{R: 200, G: 180, B: 40, A: 255}},
			{Kind: scene.OpPushClip, Rect: scene.Rect{X: 16, Y: 8, W: 88, H: 24}, Fade: [2]float32{20, 24}},
			{Kind: scene.OpFill, Rect: scene.Rect{X: 16, Y: 8, W: 88, H: 24}, Color: scene.Color{R: 230, G: 50, B: 30, A: 255}},
			{Kind: scene.OpPopClip},
		},
	}

	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()

	if err := r.draw(s); err != nil {
		t.Fatal(err)
	}
	pix, stride := r.readBack(t)
	gputest.Compare(t, "d3d11 fade", pix, stride, s)
}
