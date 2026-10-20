package widgets

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// capture returns a copy of a frame, which the tester may draw over again.
func capture(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Bounds())
	copy(out.Pix, img.Pix)
	return out
}

// at returns the pixel of a frame at (x, y) DIPs, rendered at scale 1.
func at(img *image.RGBA, x, y int) color.RGBA {
	i := img.PixOffset(x, y)
	return color.RGBA{R: img.Pix[i], G: img.Pix[i+1], B: img.Pix[i+2], A: img.Pix[i+3]}
}

// rgb converts a theme color to the pixel it paints.
func rgb(c ui.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// near reports whether two pixels are the same color, give or take tol on
// every channel, for antialiasing at an edge.
func near(got, want color.RGBA, tol int) bool {
	d := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return d(got.R, want.R) <= tol && d(got.G, want.G) <= tol && d(got.B, want.B) <= tol
}

// sameFrame reports whether two frames are pixel for pixel the same.
func sameFrame(a, b *image.RGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	return bytes.Equal(a.Pix, b.Pix)
}

// inkRightOf reports the leftmost x at or after from where a frame stops
// showing the background color, or -1 when nothing is painted there.
func inkRightOf(img *image.RGBA, from int, bg color.RGBA) int {
	b := img.Bounds()
	for x := from; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if !near(at(img, x, y), bg, 2) {
				return x
			}
		}
	}
	return -1
}

// newTester starts a tester showing view, and returns it with the theme the
// view built with.
func newTester(t *testing.T, view func(c *ui.Context), w, h int) (*ui.Tester, *ui.Theme) {
	t.Helper()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		view(c)
	}, w, h)
	if theme == nil {
		t.Fatal("the view built no theme")
	}
	return tt, theme
}
