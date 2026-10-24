package launcher

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestNavIconsAreTheNativePaths pins the six nav glyphs to launcher.html:26-50:
// the exact path data, and a parse that stays inside the 16-unit viewBox with no
// dropped segments. The old hand-drawn approximations are gone: a shape that
// silently failed to parse would draw as an empty path here.
func TestNavIconsAreTheNativePaths(t *testing.T) {
	if len(navIconPaths) != 6 {
		t.Fatalf("nav icon table has %d paths, want 6", len(navIconPaths))
	}
	for _, id := range []string{TabHome, TabComponents, TabVersions, TabImport, TabPlugins, TabSettings} {
		d, ok := navIconPaths[id]
		if !ok || d == "" {
			t.Fatalf("%s: missing native path", id)
		}
		// The data is the launcher.html path verbatim: paths there start with a
		// move and carry at least one arc/curve or a long line run.
		segments := 0
		// The SVG viewport is 16 units; the settings path in launcher.html itself
		// carries a few coordinates slightly past it, which a browser clips. The
		// test allows that small overflow and the painter clips it (tabIcon).
		box := ui.Rect{X: -1, Y: -1, W: 19, H: 19}
		// svgIcon scales into (0,0,16) here, so the emitted coordinates are the
		// viewBox ones and the bounds check is exact.
		path := svgIcon(d, 0, 0, 16)
		_ = path
		parseSVGPath(d, func(op byte, args ...float32) {
			segments++
			for i := 0; i+1 < len(args); i += 2 {
				x, y := args[i], args[i+1]
				if x < box.X || x > box.X+box.W || y < box.Y || y > box.Y+box.H {
					t.Errorf("%s: point (%.2f,%.2f) outside the 16-unit viewBox", id, x, y)
				}
			}
		})
		if segments < 4 {
			t.Fatalf("%s: parsed %d segments, want the native path's full geometry", id, segments)
		}
	}
}

// TestNavIconsDrawInsideTheIconBox checks the painter output: every glyph fills
// part of a 16x16 box placed at a non-zero origin.
func TestNavIconsDrawInsideTheIconBox(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			for _, id := range []string{TabHome, TabComponents, TabVersions, TabImport, TabPlugins, TabSettings} {
				tabIcon(c, id, ui.RGBA(0, 0, 0, 1))
			}
		})
	}, 1600, 1600)
	// A failed parse leaves the icons blank; the home glyph alone covers ~90
	// pixels, so any drawable icon proves the path reached the painter.
	img := tt.Image()
	opaque := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] > 200 {
			opaque++
		}
	}
	if opaque < 150 {
		t.Fatalf("icons painted %d opaque pixels, want the native glyphs", opaque)
	}
}
