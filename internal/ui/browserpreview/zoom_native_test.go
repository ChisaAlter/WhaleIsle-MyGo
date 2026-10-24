package browserpreview

import "testing"

// TestBrowserZoomLevelsMatchNative pins the ladder to
// reference/electron/src/main/preview-zoom.js:4-6 so it cannot drift.
func TestBrowserZoomLevelsMatchNative(t *testing.T) {
	native := []float64{0.25, 0.33, 0.5, 0.67, 0.75, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2, 2.5, 3, 4, 5}
	if len(browserZoomLevels) != len(native) {
		t.Fatalf("zoom ladder has %d steps, want %d", len(browserZoomLevels), len(native))
	}
	for i, want := range native {
		if browserZoomLevels[i] != want {
			t.Fatalf("step %d = %v, want %v", i, browserZoomLevels[i], want)
		}
	}
	// Native nextZoomLevel: in/out move within the ladder and clamp at both ends.
	if got := nextBrowserZoom(1, 1); got != 1.1 {
		t.Fatalf("zoom in from 1 = %v", got)
	}
	if got := nextBrowserZoom(1, -1); got != 0.9 {
		t.Fatalf("zoom out from 1 = %v", got)
	}
	if got := nextBrowserZoom(5, 1); got != 5 {
		t.Fatalf("zoom in at the top = %v", got)
	}
	if got := nextBrowserZoom(0.25, -1); got != 0.25 {
		t.Fatalf("zoom out at the bottom = %v", got)
	}
}
