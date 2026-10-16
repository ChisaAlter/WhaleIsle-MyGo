package darwin

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

// Pure geometry only: no AppKit startup, browser context, GUI, or screenshot.
func TestBrowserSeatOverlappingFloats(t *testing.T) {
	cases := [][]platform.Rect{
		nil,
		{{X: 2, Y: 2, Width: 5, Height: 4}, {X: 4, Y: 3, Width: 5, Height: 5}},
		{{X: -4, Y: -2, Width: 7, Height: 6}, {X: 9, Y: 9, Width: 5, Height: 5}},
		{{X: -1, Y: -1, Width: 20, Height: 20}},
		{{X: 0, Y: 0, Width: 0, Height: 5}, {X: 5, Y: 5, Width: -2, Height: 4}},
	}
	for _, holes := range cases {
		pieces := browserSeatVisible(platform.Rect{Width: 10, Height: 10}, holes)
		for y := 0; y < 10; y++ {
			for x := 0; x < 10; x++ {
				count := 0
				for _, r := range pieces {
					if x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height {
						count++
					}
				}
				visible := browserSeatContains(10, 10, float64(x)+0.5, float64(y)+0.5, holes)
				if (count == 1) != visible || count > 1 {
					t.Fatalf("holes=%v pixel=%d,%d: rendering count=%d input=%v", holes, x, y, count, visible)
				}
			}
		}
	}
	if browserSeatContains(10, 10, 10, 5, nil) || browserSeatContains(10, 10, 5, 10, nil) || browserSeatContains(10, 10, -0.1, 0, nil) {
		t.Fatal("seat boundary accepted")
	}
}
