package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBeamRasterRebuildsGeometryForDPITrackAndPalette(t *testing.T) {
	var b beamRaster
	colors := []ui.Color{ui.RGB(255, 50, 100), ui.RGB(40, 140, 255)}
	b.prepare(320, 100, 1, 2, colors, true)
	if b.width != 328 || b.height != 108 || len(b.geometry) != 328*108 {
		t.Fatal("expanded four-pixel frame geometry lost")
	}
	first := &b.geometry[0]
	b.prepare(320, 100, 1, 2, colors, true)
	if &b.geometry[0] != first {
		t.Fatal("unchanged frame reallocated raster geometry")
	}
	b.prepare(320, 100, 2, 3, colors, true)
	if b.width != 656 || b.height != 216 || b.track != 3 || &b.geometry[0] == first {
		t.Fatal("DPI/track update reused stale raster")
	}
	colors[0] = ui.RGB(0, 255, 0)
	b.prepare(320, 100, 2, 3, colors, false)
	if b.palette[0] != colors[0] || b.legacy {
		t.Fatal("custom palette retained legacy wash")
	}
	for _, g := range b.geometry {
		if g.inner < 0 || g.inner > 1 || g.outer < 0 || g.outer > 1 || g.ring < 0 || g.ring > 1 {
			t.Fatal("invalid antialias mask coverage")
		}
	}
}

func TestBeamRaster24AnglesCornerCoverageAndDarkPhase(t *testing.T) {
	const w, h float32 = 654, 193
	style := readBeamStyle(nil)
	colors := beamPalettes["legacy"]
	intervals := len(colors)
	opacity := beamOpacityTargets(style, true, 1, 1)

	var b beamRasterData
	b.prepare(w, h, 1, style.TrackWidth, colors, true)
	if b.width != int(w)+8 || b.height != int(h)+8 {
		t.Fatalf("unexpected 4px bloom shell dimensions: %dx%d", b.width, b.height)
	}

	// Sample the 2px stroke ring along each of the four 22px corners (arc midpoint
	// at radius 22 - 1 = 21 from the corner's inner circle center).
	offset := float32(22.0 - 21.0*0.70710678)
	corners := [4][2]int{
		{int(offset) + 4, int(offset) + 4},     // top-left
		{int(w-offset) + 4, int(offset) + 4},   // top-right
		{int(w-offset) + 4, int(h-offset) + 4}, // bottom-right
		{int(offset) + 4, int(h-offset) + 4},   // bottom-left
	}
	var maxAlpha [4]uint8
	minAlpha := [4]uint8{255, 255, 255, 255}

	for step := 0; step < 24; step++ {
		phase := float64(step) / 24.0
		b.render(w, h, 1, style, colors, intervals, phase, 0, false, opacity)
		for i, pt := range corners {
			a := b.pixels.RGBAAt(pt[0], pt[1]).A
			if a > maxAlpha[i] {
				maxAlpha[i] = a
			}
			if a < minAlpha[i] {
				minAlpha[i] = a
			}
		}
		// Outer 4px bloom shell bounds must remain clipped cleanly at corners.
		if cornerOutside := b.pixels.RGBAAt(0, 0).A; cornerOutside != 0 {
			t.Fatalf("step %d: pixel outside 26px outer shell leaked alpha=%d", step, cornerOutside)
		}
	}

	for i := range corners {
		if maxAlpha[i] < 100 {
			t.Fatalf("corner %d never reached high visibility across 24 angles: maxAlpha=%d", i, maxAlpha[i])
		}
		if minAlpha[i] > 40 {
			t.Fatalf("corner %d never returned to dark phase across 24 angles: minAlpha=%d", i, minAlpha[i])
		}
	}
}
