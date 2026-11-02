package pet

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestDisplayCentreFirstMatchesFixedReferenceOnEqualScale(t *testing.T) {
	displays := []mygo.Display{
		{ID: 1, Bounds: mygo.Rectangle{X: -1920, Width: 1920, Height: 1080}, ScaleFactor: 1},
		{ID: 2, Primary: true, Bounds: mygo.Rectangle{Width: 1920, Height: 1080}, ScaleFactor: 1},
	}
	for _, tc := range []struct {
		x, y int
		want int64
	}{{-121, 100, 1}, {-120, 100, 2}, {-100, 100, 2}, {100, 100, 2}, {-2000, -1000, 2}} {
		for _, size := range []int{1, 144, 240, 360, 384, 10000} {
			stage := mygo.Rectangle{X: tc.x, Y: tc.y, Width: size, Height: size}
			if got := activePetDisplay(displays, stage); got.ID != tc.want {
				t.Fatalf("x=%d y=%d size=%d display=%d want=%d", tc.x, tc.y, size, got.ID, tc.want)
			}
		}
	}
	// Centre in a display gap falls back to the actual anchor monitor.
	displays[1].Bounds.X = 500
	if got := activePetDisplay(displays, mygo.Rectangle{X: -50, Y: 100}); got.ID != 1 {
		t.Fatalf("gap fallback=%d", got.ID)
	}
}
