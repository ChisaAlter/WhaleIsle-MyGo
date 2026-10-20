package ui

import (
	"math"
	"testing"
)

// pixelHost is a headless host that reports its size in device pixels, as a
// window's surface does (host.pixels). A test can then cover the DIP
// normalization runFrame applies on a scaled display without opening a window;
// the rendering itself is the same in-memory one.
type pixelHost struct{ *headless }

func (h pixelHost) pixels() bool { return true }

// newPixelTester builds a tester whose host reports device pixels, at scale
// device pixels per DIP.
func newPixelTester(view func(c *Context), width, height int, scale float32) *Tester {
	h := &headless{w: float32(width), h: float32(height), scale: scale}
	t := &Tester{rt: newRuntime(func(c *context) { view(makeContext(c)) }, pixelHost{h}), h: h}
	t.rt.collect = true
	t.rt.strict = true
	t.rt.handleChecks = true
	t.settle()
	return t
}

// scaleCase is a window size the backend reported, and the DIP and device pixel
// geometry runFrame must derive from it.
type scaleCase struct {
	name       string
	scale      float32
	pixels     [2]int     // size() as the window's surface reports it
	wantDIP    [2]float32 // the room the view lays out in
	wantDevice [2]int     // the surface it paints
}

// scaledWindows are the display scales a window can report. 150% was checked on
// a real display; 125% and 200% are covered here, through the fake pixel host.
var scaledWindows = []scaleCase{
	{"100 percent", 1, [2]int{1600, 1000}, [2]float32{1600, 1000}, [2]int{1600, 1000}},
	{"125 percent", 1.25, [2]int{1600, 1000}, [2]float32{1280, 800}, [2]int{1600, 1000}},
	{"150 percent", 1.5, [2]int{1600, 1000}, [2]float32{1066.6667, 666.6667}, [2]int{1600, 1000}},
	{"200 percent", 2, [2]int{1600, 1000}, [2]float32{800, 500}, [2]int{1600, 1000}},
}

// TestPixelHostNormalizesDevicePixelsToDIPs is the scaled-window case: the host
// reports device pixels, the view lays out in DIPs, paint multiplies them back
// by the scale, and the painted size stays the window's device pixels.
func TestPixelHostNormalizesDevicePixelsToDIPs(t *testing.T) {
	for _, tc := range scaledWindows {
		t.Run(tc.name, func(t *testing.T) {
			tt := newPixelTester(func(c *Context) {
				Box(c).Fill().Label("root")
			}, tc.pixels[0], tc.pixels[1], tc.scale)
			root, ok := tt.Find("root")
			if !ok {
				t.Fatal("the root did not build")
			}
			if !sameLength(root.W, tc.wantDIP[0]) || !sameLength(root.H, tc.wantDIP[1]) {
				t.Fatalf("layout %.2f×%.2f DIP, want %.2f×%.2f: the view laid out the pixel size",
					root.W, root.H, tc.wantDIP[0], tc.wantDIP[1])
			}
			if got, want := tt.rt.painted, [3]float32{float32(tc.pixels[0]), float32(tc.pixels[1]), tc.scale}; got != want {
				t.Fatalf("painted = %v, want %v: the window's device pixels and scale", got, want)
			}
			if got := tt.rt.scene.Scale; got != tc.scale {
				t.Fatalf("paint scale = %v, want %v", got, tc.scale)
			}
			if b := tt.Image().Bounds(); b.Dx() != tc.wantDevice[0] || b.Dy() != tc.wantDevice[1] {
				t.Fatalf("surface %d×%d pixels, want %d×%d", b.Dx(), b.Dy(), tc.wantDevice[0], tc.wantDevice[1])
			}
		})
	}
}

// TestHeadlessHostKeepsDIPs is the other side of the normalization: a host that
// renders into memory reports DIPs already, so the size stands as it is and
// only paint scales.
func TestHeadlessHostKeepsDIPs(t *testing.T) {
	for _, tc := range scaledWindows {
		t.Run(tc.name, func(t *testing.T) {
			tt := NewTester(func(c *Context) {
				Box(c).Fill().Label("root")
			}, tc.pixels[0], tc.pixels[1])
			tt.SetScale(tc.scale)
			root, ok := tt.Find("root")
			if !ok {
				t.Fatal("the root did not build")
			}
			if !sameLength(root.W, float32(tc.pixels[0])) || !sameLength(root.H, float32(tc.pixels[1])) {
				t.Fatalf("layout %.2f×%.2f DIP, want the host size %d×%d unchanged",
					root.W, root.H, tc.pixels[0], tc.pixels[1])
			}
			if got, want := tt.rt.painted, [3]float32{float32(tc.pixels[0]), float32(tc.pixels[1]), tc.scale}; got != want {
				t.Fatalf("painted = %v, want %v", got, want)
			}
			// The surface still holds device pixels: DIPs × scale.
			wantW := int(math.Ceil(float64(float32(tc.pixels[0]) * tc.scale)))
			wantH := int(math.Ceil(float64(float32(tc.pixels[1]) * tc.scale)))
			if b := tt.Image().Bounds(); b.Dx() != wantW || b.Dy() != wantH {
				t.Fatalf("surface %d×%d pixels, want %d×%d", b.Dx(), b.Dy(), wantW, wantH)
			}
		})
	}
}

// TestScaleOneIsIdentityForBothHosts pins 100%: a pixel host and a headless one
// of the same numbers lay out the same room and paint the same surface.
func TestScaleOneIsIdentityForBothHosts(t *testing.T) {
	build := func(t *testing.T, pixel bool) (root Rect, painted [3]float32, surface [2]int) {
		t.Helper()
		view := func(c *Context) { Box(c).Fill().Label("root") }
		var tt *Tester
		if pixel {
			tt = newPixelTester(view, 1600, 1000, 1)
		} else {
			tt = NewTester(view, 1600, 1000)
		}
		r, ok := tt.Find("root")
		if !ok {
			t.Fatal("the root did not build")
		}
		b := tt.Image().Bounds()
		return r, tt.rt.painted, [2]int{b.Dx(), b.Dy()}
	}
	pixelRect, pixelPainted, pixelSurface := build(t, true)
	headlessRect, headlessPainted, headlessSurface := build(t, false)
	if pixelRect != headlessRect {
		t.Fatalf("layout differs at 100%%: pixel host %+v, headless %+v", pixelRect, headlessRect)
	}
	if pixelPainted != headlessPainted {
		t.Fatalf("painted differs at 100%%: %v vs %v", pixelPainted, headlessPainted)
	}
	if pixelSurface != headlessSurface {
		t.Fatalf("surface differs at 100%%: %v vs %v", pixelSurface, headlessSurface)
	}
	if pixelRect.W != 1600 || pixelRect.H != 1000 {
		t.Fatalf("layout at 100%% = %v, want 1600×1000 DIP", pixelRect)
	}
}

// TestNonPositiveScaleFallsBackToOne covers a backend that reports no scale: the
// frame must treat it as 100%, never divide by zero, and keep the geometry a
// 100% window has.
func TestNonPositiveScaleFallsBackToOne(t *testing.T) {
	for _, scale := range []float32{0, -2} {
		t.Run("pixel host at "+formatScale(scale), func(t *testing.T) {
			tt := newPixelTester(func(c *Context) { Box(c).Fill().Label("root") }, 1600, 1000, scale)
			root, ok := tt.Find("root")
			if !ok {
				t.Fatal("the root did not build")
			}
			if !sameLength(root.W, 1600) || !sameLength(root.H, 1000) {
				t.Fatalf("layout %.2f×%.2f DIP, want 1600×1000: the scale should fall back to 1", root.W, root.H)
			}
			if got, want := tt.rt.painted, [3]float32{1600, 1000, 1}; got != want {
				t.Fatalf("painted = %v, want %v", got, want)
			}
			if b := tt.Image().Bounds(); b.Dx() != 1600 || b.Dy() != 1000 {
				t.Fatalf("surface %d×%d pixels, want 1600×1000", b.Dx(), b.Dy())
			}
		})
		t.Run("headless at "+formatScale(scale), func(t *testing.T) {
			tt := NewTester(func(c *Context) { Box(c).Fill().Label("root") }, 1600, 1000)
			tt.SetScale(scale)
			root, ok := tt.Find("root")
			if !ok {
				t.Fatal("the root did not build")
			}
			if !sameLength(root.W, 1600) || !sameLength(root.H, 1000) {
				t.Fatalf("layout %.2f×%.2f DIP, want 1600×1000", root.W, root.H)
			}
			if got, want := tt.rt.painted, [3]float32{1600, 1000, 1}; got != want {
				t.Fatalf("painted = %v, want %v", got, want)
			}
		})
	}
}

// TestPaintsDIPsAtDevicePixels checks what the scale multiplies: a 20 DIP square
// at (10, 10) covers device pixels 10..30 at 100%, 12.5..37.5 at 125% and
// 20..60 at 200%.
func TestPaintsDIPsAtDevicePixels(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		scale                   float32
		inside, outside, before [2]int
	}{
		{"100 percent", 1, [2]int{20, 20}, [2]int{34, 34}, [2]int{8, 8}},
		{"125 percent", 1.25, [2]int{20, 20}, [2]int{40, 40}, [2]int{10, 10}},
		{"200 percent", 2, [2]int{25, 25}, [2]int{65, 65}, [2]int{12, 12}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := newPixelTester(func(c *Context) {
				Box(c).Fill().Background(RGB(255, 255, 255))
				Box(c).Absolute().Left(10).Top(10).Width(20).Height(20).Background(RGB(255, 0, 0))
			}, 200, 200, tc.scale)
			image := tt.Image()
			red := func(p [2]int) bool {
				c := image.RGBAAt(p[0], p[1])
				return c.R > 200 && c.G < 60 && c.B < 60
			}
			if !red(tc.inside) {
				t.Fatalf("the square is missing at %v", tc.inside)
			}
			if red(tc.outside) || red(tc.before) {
				t.Fatalf("the square spilled to %v or %v: DIPs are not being painted at scale %v",
					tc.outside, tc.before, tc.scale)
			}
		})
	}
}

// sameLength reports whether two lengths are the same to within the rounding a layout
// does.
func sameLength(a, b float32) bool {
	d := a - b
	return d < .5 && d > -.5
}

// formatScale names a scale in a test's name.
func formatScale(scale float32) string {
	if scale == 0 {
		return "zero"
	}
	return "negative"
}
