package browserpreview

import (
	"math"
	"testing"
)

func TestBrowserResizeFixture(t *testing.T) {
	for _, tc := range []struct {
		dir          string
		dx, dy       float64
		wantW, wantH int
	}{
		{"east", 10, 0, 420, 300}, {"west", 10, 0, 380, 300}, {"south", 0, 10, 400, 320},
		{"southeast", 10, 10, 420, 320}, {"southwest", 10, 10, 380, 320},
	} {
		w, h := resizeBrowserRail(400, 300, tc.dx, tc.dy, 800, 600, 1, tc.dir, 0)
		if w != tc.wantW || h != tc.wantH {
			t.Fatalf("%s = %dx%d", tc.dir, w, h)
		}
	}
	w, h := resizeBrowserRail(400, 300, 10, 0, 400, 300, 0.5, "east", 0)
	if w != 440 || h != 300 {
		t.Fatalf("fit/zoom translation = %dx%d", w, h)
	}
	for _, dir := range []string{"east", "west", "south", "southwest", "southeast"} {
		for _, ratio := range []float64{0, 375.0 / 667.0, 667.0 / 375.0} {
			for _, delta := range []float64{-1e6, -10, 0, 10, 1e6, math.NaN(), math.Inf(1)} {
				w, h := resizeBrowserRail(375, 667, delta, delta, 780, 590, 1, dir, ratio)
				if !validBrowserDimensions(w, h) {
					t.Fatalf("invalid %s ratio %g delta %g: %dx%d", dir, ratio, delta, w, h)
				}
				if ratio > 0 && math.Abs(float64(w)/float64(h)-ratio) > 0.01 {
					t.Fatalf("ratio lost: %dx%d", w, h)
				}
			}
		}
	}
	x, y, w, h, scale := browserDeviceLayout(800, 600, 375, 667, 1)
	if x < 10 || y < 0 || x+w+10 > 800 || y+h+10 > 600 || scale <= 0 {
		t.Fatalf("rails overlap guest: %d %d %d %d %g", x, y, w, h, scale)
	}
	_, _, w, h, _ = browserDeviceLayout(20, 10, 375, 667, 1)
	if w != 0 || h != 0 {
		t.Fatal("tiny pane must hide device")
	}
}
