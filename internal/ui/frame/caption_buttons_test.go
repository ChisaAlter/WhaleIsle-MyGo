package frame

import (
	"image"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// over composites top over bottom, as the painter's source-over fill does.
func over(top, bottom ui.Color) ui.Color {
	a := float32(top.A) / 255
	mix := func(t, b uint8) uint8 {
		return uint8(float32(t)*a + float32(b)*(1-a) + .5)
	}
	return ui.RGBA(mix(top.R, bottom.R), mix(top.G, bottom.G), mix(top.B, bottom.B), 1)
}

func samePixel(got color.RGBA, want ui.Color, tolerance int) bool {
	d := func(a uint8, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return d(got.R, want.R) <= tolerance && d(got.G, want.G) <= tolerance && d(got.B, want.B) <= tolerance
}

// captionHit boxes mirror the native geometry: 32x32 at Top 12, right offsets
// 8 (close), 40 (maximize), 72 (minimize).
func captionHit(windowWidth, right int) image.Rectangle {
	x := windowWidth - right - captionButtonSize
	return image.Rect(x, captionButtonTop, x+captionButtonSize, captionButtonTop+captionButtonSize)
}

// fillPoint is inside a button's rounded fill but clear of its glyph.
func fillPoint(r image.Rectangle) image.Point {
	return image.Pt(r.Min.X+3, r.Max.Y-3)
}

// glyphInk counts glyph pixels in a button's inner box: dark ink normally, pure
// white on the close button's hover fill. The 4 DIP inset keeps the rounded
// border out; thin 1 DIP strokes are antialiased, so "ink" is anything clearly
// darker than the page, and "white" is brighter than the page itself.
func glyphInk(img *image.RGBA, r image.Rectangle) (dark, white int) {
	for y := r.Min.Y + 4; y < r.Max.Y-4; y++ {
		for x := r.Min.X + 4; x < r.Max.X-4; x++ {
			p := img.RGBAAt(x, y)
			if p.R < 200 && p.G < 200 && p.B < 200 {
				dark++
			}
			if p.R > 250 && p.G > 250 && p.B > 250 {
				white++
			}
		}
	}
	return dark, white
}

// TestCaptionButtonsHoverFills drives the pointer over each control and samples
// the frame: normal state stays transparent with an ink glyph, close fills
// #e81123 with a white glyph, and minimize/maximize take the interactive hover
// background (window-controls.css:39-46, harness-chrome-inject.js:116-122).
func TestCaptionButtonsHoverFills(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)

	const (
		winW = 400
		winH = 200
	)
	page := ui.RGB(250, 250, 250)
	ink := ui.RGB(40, 40, 40)
	closeFill := ui.RGB(0xe8, 0x11, 0x23)
	var hoverFill ui.Color
	var closed, minimized, maximizedCalls int
	view := func(c *ui.Context) {
		hoverFill = c.Theme().SurfaceHover
		ui.Column(c).Fill().Background(page).Children(func() {
			drawCaptionButtons(c, ink, captionWindowActions{
				Close:          func() { closed++ },
				Minimize:       func() { minimized++ },
				ToggleMaximize: func() { maximizedCalls++ },
				Maximized:      func() bool { return false },
			})
		})
	}
	tt := ui.NewTester(view, winW, winH)
	tt.Frame()

	hit := func(name string) image.Rectangle {
		t.Helper()
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("caption button %q is missing; texts = %q", name, tt.Texts())
		}
		return image.Rect(int(r.X), int(r.Y), int(r.X+r.W), int(r.Y+r.H))
	}
	closeHit, maxHit, minHit := hit("关闭"), hit("最大化"), hit("最小化")
	// Geometry stays the native one, in the reference's right-to-left order.
	wantClose, wantMax, wantMin := captionHit(winW, 8), captionHit(winW, 40), captionHit(winW, 72)
	if closeHit != wantClose || maxHit != wantMax || minHit != wantMin {
		t.Fatalf("caption boxes = close %v max %v min %v, want %v %v %v", closeHit, maxHit, minHit, wantClose, wantMax, wantMin)
	}

	// 1. Normal state: transparent buttons, ink glyph, no fill anywhere.
	img := tt.Image()
	for name, hit := range map[string]image.Rectangle{"close": closeHit, "maximize": maxHit, "minimize": minHit} {
		if got := img.RGBAAt(fillPoint(hit).X, fillPoint(hit).Y); !samePixel(got, page, 1) {
			t.Fatalf("normal %s painted a fill: %v on %v", name, got, page)
		}
		if dark, _ := glyphInk(img, hit); dark == 0 {
			t.Fatalf("normal %s lost its ink glyph", name)
		}
	}

	// 2. Close hover: #e81123 fill and a white glyph.
	tt.Move(float32(closeHit.Min.X+16), float32(closeHit.Min.Y+16))
	tt.Frame()
	img = tt.Image()
	if got := img.RGBAAt(fillPoint(closeHit).X, fillPoint(closeHit).Y); !samePixel(got, closeFill, 1) {
		t.Fatalf("close hover fill = %v, want #e81123", got)
	}
	if _, white := glyphInk(img, closeHit); white == 0 {
		t.Fatal("close hover did not switch to a white glyph")
	}
	for name, hit := range map[string]image.Rectangle{"maximize": maxHit, "minimize": minHit} {
		if got := img.RGBAAt(fillPoint(hit).X, fillPoint(hit).Y); !samePixel(got, page, 1) {
			t.Fatalf("hovering close tinted %s: %v", name, got)
		}
	}

	// 3. Minimize and maximize hover: the interactive hover background over the
	// page, glyph stays in ink.
	wantTint := over(hoverFill, page)
	for name, hit := range map[string]image.Rectangle{"minimize": minHit, "maximize": maxHit} {
		tt.Move(float32(hit.Min.X+16), float32(hit.Min.Y+16))
		tt.Frame()
		img = tt.Image()
		if got := img.RGBAAt(fillPoint(hit).X, fillPoint(hit).Y); !samePixel(got, wantTint, 1) {
			t.Fatalf("%s hover fill = %v, want the interactive tint %v", name, got, wantTint)
		}
		if dark, _ := glyphInk(img, hit); dark == 0 {
			t.Fatalf("%s hover lost its ink glyph", name)
		}
		other := closeHit
		if name == "minimize" {
			other = maxHit
		}
		if got := img.RGBAAt(fillPoint(other).X, fillPoint(other).Y); !samePixel(got, page, 1) {
			t.Fatalf("hovering %s tinted another button: %v", name, got)
		}
	}

	// 4. Moving away clears the fills again.
	tt.Move(winW/2, winH-10)
	tt.Frame()
	img = tt.Image()
	for name, hit := range map[string]image.Rectangle{"close": closeHit, "maximize": maxHit, "minimize": minHit} {
		if got := img.RGBAAt(fillPoint(hit).X, fillPoint(hit).Y); !samePixel(got, page, 1) {
			t.Fatalf("%s kept a hover fill after the pointer left: %v", name, got)
		}
	}

	// 5. The hover fill never steals the click: each control still runs.
	for _, hit := range []image.Rectangle{closeHit, maxHit, minHit} {
		tt.ClickAt(float32(hit.Min.X+16), float32(hit.Min.Y+16))
		tt.Frame()
	}
	if closed != 1 || minimized != 1 || maximizedCalls != 1 {
		t.Fatalf("caption clicks = close %d, minimize %d, maximize %d", closed, minimized, maximizedCalls)
	}
}

// TestCaptionButtonsMaximizedGlyph keeps the restore glyph on the maximize
// control, which the hover work must not disturb.
func TestCaptionButtonsMaximizedGlyph(t *testing.T) {
	page := ui.RGB(250, 250, 250)
	ink := ui.RGB(40, 40, 40)
	maximized := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(page).Children(func() {
			drawCaptionButtons(c, ink, captionWindowActions{Maximized: func() bool { return maximized }})
		})
	}, 400, 200)
	tt.Frame()
	maxHit := captionHit(400, 40)
	// The normal glyph is one 10x10 outline; the restore glyph is two offset
	// outlines, so it covers more ink.
	count := func() int {
		img := tt.Image()
		n := 0
		for y := maxHit.Min.Y; y < maxHit.Max.Y; y++ {
			for x := maxHit.Min.X; x < maxHit.Max.X; x++ {
				if p := img.RGBAAt(x, y); p.R < 200 && p.G < 200 && p.B < 200 {
					n++
				}
			}
		}
		return n
	}
	normal := count()
	maximized = true
	tt.Frame()
	restore := count()
	if restore <= normal {
		t.Fatalf("restore glyph ink = %d, want more than the %d of the maximized glyph", restore, normal)
	}
}
