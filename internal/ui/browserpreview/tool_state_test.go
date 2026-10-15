package browserpreview

import (
	"math"
	"testing"
)

func TestBrowserToolStateFixture(t *testing.T) {
	for _, p := range browserDevicePresets {
		if !validBrowserDimensions(p.Width, p.Height) {
			t.Fatalf("invalid preset: %+v", p)
		}
	}
	for _, size := range [][2]int{{0, 0}, {239, 667}, {375, 3841}, {3840, 3840}} {
		if validBrowserDimensions(size[0], size[1]) {
			t.Fatalf("accepted unsafe size %v", size)
		}
	}
	if nextBrowserZoom(1, 1) != 1.1 || nextBrowserZoom(1, -1) != 0.9 || nextBrowserZoom(0.25, -1) != 0.25 || nextBrowserZoom(5, 1) != 5 {
		t.Fatal("zoom step mismatch")
	}
	if nextBrowserZoom(math.NaN(), 1) != 1.1 || nextBrowserZoom(math.Inf(1), -1) != 0.9 {
		t.Fatal("nonfinite zoom not normalized")
	}
	x, y, w, h, scale := fitBrowserDevice(800, 600, 375, 667, 1)
	if x < 0 || y < 0 || w > 800 || h > 600 || w < 1 || h < 1 || scale >= 1 {
		t.Fatalf("bad fitted device %d %d %d %d %g", x, y, w, h, scale)
	}
	_, _, w, h, _ = fitBrowserDevice(300, 200, 1280, 800, 5)
	if w > 300 || h > 200 {
		t.Fatal("zoom escaped panel")
	}
	_, _, w, h, _ = fitBrowserDevice(0, 0, 375, 667, 1)
	if w != 0 || h != 0 {
		t.Fatal("zero seat must stay hidden")
	}
	state := browserToolSettings{Width: 375, Height: 667, AspectLocked: true}
	state.rotate()
	if state.Width != 667 || state.Height != 375 || !state.AspectLocked {
		t.Fatal("rotation lost ratio lock")
	}
	width, height, err := state.dimensions("1000", "375")
	if err != nil || width != 1000 || height != 562 {
		t.Fatalf("locked width: %d %d %v", width, height, err)
	}
	if _, _, err = state.dimensions("NaN", "375"); err == nil {
		t.Fatal("accepted NaN input")
	}
	a := browserToolSettings{Width: 375, Height: 667}
	b := a
	b.rotate()
	if a.Width != 375 || a.Height != 667 {
		t.Fatal("settings leaked across tabs")
	}
}
