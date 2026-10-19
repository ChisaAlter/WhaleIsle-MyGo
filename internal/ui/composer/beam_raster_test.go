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
