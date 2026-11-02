package chrome

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestMenuSurfaceRoundsItsOwnElement pins that the radius handed to MenuSurface
// is the panel's own corner, not only the glass's. Most popover callers build
// the panel without a .Radius(...): the element then kept square corners, and
// its border and shadow were drawn square around the rounded glass, which read
// as a second frame outside the menu's rounded edge.
func TestMenuSurfaceRoundsItsOwnElement(t *testing.T) {
	const x, y, w, h, radius = 20, 20, 120, 60, 12
	border := ui.RGB(255, 0, 0) // a hue no shadow or fill can produce
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255))
		MenuSurface(ui.Box(c).Absolute().Left(x).Top(y).Width(w).Height(h), ui.RGB(30, 30, 30), border, radius)
	}, 180, 120)
	tt.Frame()
	img := tt.Image()

	// Pixels in the corner square that lie outside the rounded arc: nothing of the
	// panel may be there. A square border puts its hairline on the very corner.
	for _, p := range [][2]int{{x, y}, {x + 1, y + 1}, {x + w - 1, y}, {x + w - 2, y + 1}, {x, y + h - 1}, {x + 1, y + h - 2}, {x + w - 1, y + h - 1}, {x + w - 2, y + h - 2}} {
		got := img.RGBAAt(p[0], p[1])
		if got.R != got.G || got.G != got.B {
			t.Fatalf("pixel %v outside the rounded corner is tinted by the border: %+v", p, got)
		}
	}

	// The hairline itself must survive along the straight edges.
	if got := img.RGBAAt(x+w/2, y); got.R <= got.G {
		t.Fatalf("the menu lost its hairline on the top edge: %+v", got)
	}
}
