package composer

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestBeamFiniteFrameJobExchangesBuffersAndRejectsOldGeometry(t *testing.T) {
	var b beamRaster
	style := readBeamStyle(nil)
	colors := []ui.Color{ui.RGB(255, 50, 100), ui.RGB(40, 140, 255)}
	width, phase, paintOne := float32(80), 0.0, true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Width(width).Height(32).Draw(func(p *ui.Painter, r ui.Rect) {
			if paintOne {
				paintOne = false
				b.paint(p, r, style, colors, 2, phase, 0, false, beamOpacities{.6, .42, .36})
			}
		})
	}, 120, 60)
	await := func() *beamRasterData {
		t.Helper()
		if !b.pending || cap(b.frames) != 1 {
			t.Fatal("Beam did not retain one bounded image job")
		}
		select {
		case frame := <-b.frames:
			return frame
		case <-time.After(5 * time.Second):
			t.Fatal("finite Beam job did not return")
			return nil
		}
	}
	// Return an actually computed result to native paint; no numerical or
	// presentation worker is substituted by this scheduling fixture.
	first := await()
	b.frames <- first
	paintOne = true
	tt.Frame()
	second := await()
	b.frames <- second
	phase = .25
	paintOne = true
	tt.Frame()
	front := b.front
	snapshot := bytes.Clone(front.Pix)
	third := await()
	if third.pixels == front || !bytes.Equal(front.Pix, snapshot) {
		t.Fatal("worker overwrote the RGBA owned by native paint")
	}
	if bytes.Equal(third.pixels.Pix, snapshot) {
		t.Fatal("quarter-turn frame did not update the actual raster")
	}
	b.frames <- third
	width = 100
	paintOne = true
	tt.Frame()
	if b.front != front {
		t.Fatal("obsolete width replaced the last complete front buffer")
	}
	fourth := await()
	b.frames <- fourth
	paintOne = true
	tt.Frame()
	if b.front == front || b.front.Rect.Dx() != 108 {
		t.Fatal("current resize result was not adopted")
	}
	await() // Finish the last finite job; no waiting goroutine remains.
}
