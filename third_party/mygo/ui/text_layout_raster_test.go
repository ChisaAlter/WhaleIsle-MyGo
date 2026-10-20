//go:build (windows || darwin || linux) && (amd64 || arm64)

package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"math"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

// Draw through the ordinary shared-atlas Painter into a generously sized
// transparent canvas. This reference does not use Rasterize's bounds helper or
// its private-system seam, and catches missing ink, changed hinting and spans.
func textRasterNativeReference(t *testing.T, layout *TextLayout, color Color, scale float32) *image.RGBA {
	t.Helper()
	system := text.Shared()
	system.BeginFrame()
	var display scene.Scene
	paint := func() {
		display.Reset(int(640*scale), int(160*scale), scene.Color{})
		display.Scale, display.Text = scale, system.TextParams()
		display.MaskAtlas, display.ColorAtlas = system.MaskAtlas, system.ColorAtlas
		painter := Painter{rt: &engine{text: system}, s: &display, scale: scale, opacity: 1, clip: Rect{0, 0, 640, 160}, opaque: false}
		painter.TextLayout(layout, 24, 24, color)
	}
	paint()
	for attempt := 0; attempt < 2 && system.Full(); attempt++ {
		system.MakeRoom()
		paint()
	}
	if system.Full() {
		t.Fatal("native reference exhausted its glyph atlas")
	}
	pixels := raster.NewImage(display.Width, display.Height)
	raster.Render(pixels, &display)
	return &image.RGBA{Pix: pixels.RGBA(), Stride: display.Width * 4, Rect: image.Rect(0, 0, display.Width, display.Height)}
}

func textRasterInk(img *image.RGBA) image.Rectangle {
	var bounds image.Rectangle
	if img != nil {
		for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
			for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
				if img.RGBAAt(x, y).A != 0 {
					bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}
	}
	return bounds
}

func textRasterMatchesPainter(t *testing.T, layout *TextLayout, color Color, scale float32) (*image.RGBA, Point) {
	t.Helper()
	got, offset := layout.Rasterize(color, scale)
	if got == nil || textRasterInk(got).Empty() {
		t.Fatal("real native text produced no raster ink")
	}
	want := textRasterNativeReference(t, layout, color, scale)
	placed := image.NewRGBA(want.Bounds())
	origin := image.Pt(int(math.Round(float64((24+offset.X)*scale))), int(math.Round(float64((24+offset.Y)*scale))))
	box := got.Bounds().Add(origin)
	if !box.In(placed.Bounds()) {
		t.Fatalf("test canvas does not contain bitmap at offset %+v", offset)
	}
	draw.Draw(placed, box, got, got.Bounds().Min, draw.Over)
	if !bytes.Equal(placed.Pix, want.Pix) {
		t.Fatalf("native raster differs from ordinary Painter at DPI scale %v, DIP offset %+v", scale, offset)
	}
	ink := textRasterInk(got)
	if ink.Min.X <= 0 || ink.Min.Y <= 0 || ink.Max.X >= got.Rect.Max.X || ink.Max.Y >= got.Rect.Max.Y {
		t.Fatalf("ink touches bitmap boundary: ink=%v bitmap=%v", ink, got.Rect)
	}
	return got, offset
}

func TestTextLayoutRasterNativeDPIAndTransparency(t *testing.T) {
	layout := ShapeTextLine("Native 鲸屿 אבג e\u0301", Font{Size: 24}, 0)
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			img, _ := textRasterMatchesPainter(t, layout, RGB(255, 255, 255).Alpha(0.5), scale)
			partial := false
			for i := 0; i < len(img.Pix); i += 4 {
				p := img.Pix[i:][:4]
				if p[0] != p[1] || p[1] != p[2] || p[2] != p[3] {
					t.Fatal("white transparent glyph used screen-subpixel coverage or non-premultiplied channels")
				}
				partial = partial || p[3] > 0 && p[3] < 255
			}
			if !partial {
				t.Fatal("transparent native foreground became opaque")
			}
		})
	}
	img, _ := layout.Rasterize(RGB(220, 70, 30).Alpha(0.5), 1.5)
	if img == nil {
		t.Fatal("transparent colored foreground produced no native bitmap")
	}
	for i := 0; i < len(img.Pix); i += 4 {
		p := img.Pix[i:][:4]
		if p[0] > p[3] || p[1] > p[3] || p[2] > p[3] {
			t.Fatal("colored transparent foreground is not premultiplied")
		}
	}
}

func TestShapeTextLineNativeFallbackAndGraphemeEllipsis(t *testing.T) {
	if err := text.Shared().RegisterFont(goregular.TTF, "TextRaster ASCII"); err != nil {
		t.Fatal(err)
	}
	fallback := ShapeTextLine("A鲸", Font{Family: "TextRaster ASCII", Size: 24}, 0)
	var latinFont, cjkFont *text.Font
	for _, glyph := range fallback.l.Lines[0].Glyphs {
		if glyph.Cluster == 0 {
			latinFont = glyph.Font
		}
		if glyph.Cluster == 1 && glyph.ID != 0 {
			cjkFont = glyph.Font
		}
	}
	if latinFont == nil || cjkFont == nil || latinFont == cjkFont {
		t.Fatal("ASCII-only registered font did not use real system CJK fallback")
	}
	textRasterMatchesPainter(t, fallback, RGB(20, 60, 90), 1.5)
	rtl := ShapeTextLine("שלום אבג xyz", Font{Size: 24}, 0)
	if len(rtl.l.Lines) != 1 || !rtl.l.Lines[0].RTL {
		t.Fatal("native single-line shaping lost RTL paragraph direction")
	}
	textRasterMatchesPainter(t, rtl, RGB(20, 60, 90), 1.25)
	for _, width := range []float32{24, 48, 72, 120} {
		line := ShapeTextLine("e\u0301 👩‍💻 🇨🇳 אבג long remaining text", Font{Size: 24}, width)
		if len(line.l.Lines) != 1 || !line.l.Truncated || line.l.Lines[0].Width > width+0.01 {
			t.Fatalf("width %v did not use the native one-line ellipsis", width)
		}
		end := line.l.Lines[0].End
		if !slices.Contains(line.boundaries, end) {
			t.Fatalf("ellipsis cut combining/surrogate/ZWJ/flag cluster at rune %d", end)
		}
		marker := false
		for _, glyph := range line.l.Lines[0].Glyphs {
			marker = marker || glyph.Runes == 0 && glyph.Cluster == end
		}
		if !marker {
			t.Fatal("truncated native line lost its ellipsis glyph")
		}
		textRasterMatchesPainter(t, line, RGB(255, 255, 255), 1)
	}
}

func TestTextLayoutRasterPreservesItalicInkAndRichPaint(t *testing.T) {
	if err := text.Shared().RegisterFont(goitalic.TTF, "TextRaster Italic"); err != nil {
		t.Fatal(err)
	}
	italic := ShapeTextLine("jfÁ", Font{Family: "TextRaster Italic", Italic: true, Size: 48}, 0)
	img, offset := textRasterMatchesPainter(t, italic, RGB(200, 30, 20), 2)
	ink := textRasterInk(img)
	width, _ := italic.Size()
	if float32(ink.Min.X)/2+offset.X >= 0 && float32(ink.Max.X)/2+offset.X <= width {
		t.Fatal("fixture did not exercise actual italic overhang outside logical advances")
	}
	spans := []Span{
		{Text: "RED", Color: RGB(255, 0, 0), Background: RGB(255, 220, 0), Size: 32, Weight: 700},
		{Text: " lines", Color: RGB(0, 0, 255), Underline: true, Strikethrough: true, DecorationColor: RGB(0, 255, 0), DecorationThickness: 2},
		{Text: " wave", Color: RGB(0, 0, 255), WavyUnderline: true, DecorationColor: RGB(0, 255, 0)},
	}
	rich := ShapeRichText(spans, Font{Size: 24}, 0)
	painted, _ := textRasterMatchesPainter(t, rich, RGB(255, 255, 255), 1.5)
	red, blue, yellow, green := 0, 0, 0, 0
	for i := 0; i < len(painted.Pix); i += 4 {
		p := painted.Pix[i:][:4]
		if p[3] < 200 {
			continue
		}
		if p[0] > 200 && p[1] < 50 && p[2] < 50 {
			red++
		}
		if p[2] > 200 && p[0] < 50 && p[1] < 50 {
			blue++
		}
		if p[0] > 200 && p[1] > 170 && p[2] < 50 {
			yellow++
		}
		if p[1] > 200 && p[0] < 50 && p[2] < 50 {
			green++
		}
	}
	if min(red, blue, yellow, green) < 8 {
		t.Fatalf("lost rich span foreground/background/decorations: red=%d blue=%d yellow=%d green=%d", red, blue, yellow, green)
	}
}

func TestTextLayoutRasterRetainsNativeColorGlyph(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("native emoji color-font availability varies on Linux, as in internal/text TestColorEmoji")
	}
	layout := ShapeTextLine("🎉", Font{Size: 32}, 0)
	img, _ := textRasterMatchesPainter(t, layout, RGB(255, 255, 255), 1.5)
	colored := false
	for i := 0; i < len(img.Pix); i += 4 {
		p := img.Pix[i:][:4]
		colored = colored || p[3] > 100 && (p[0] != p[1] || p[1] != p[2])
		if max(p[0], p[1], p[2]) > p[3] {
			t.Fatal("native color glyph lost premultiplied alpha")
		}
	}
	if !colored {
		t.Fatal("real native color glyph was replaced by a foreground mask")
	}
}

func TestTextLayoutRasterEmptyAndInvalidScale(t *testing.T) {
	for _, layout := range []*TextLayout{nil, {}, ShapeText("", Font{Size: 24}, 0), ShapeText("   \n", Font{Size: 24}, 0)} {
		if img, offset := layout.Rasterize(RGB(255, 255, 255), 1); img != nil || offset != (Point{}) {
			t.Fatal("empty layout allocated a fake visible bitmap")
		}
	}
	layout := ShapeText("native", Font{Size: 24}, 0)
	for _, scale := range []float32{0, -1, float32(math.NaN()), float32(math.Inf(1))} {
		if img, _ := layout.Rasterize(RGB(255, 255, 255), scale); img != nil {
			t.Fatal("invalid DPI scale produced a bitmap")
		}
	}
}

func TestTextLayoutRasterConcurrentWithSharedPainter(t *testing.T) {
	layout := ShapeTextLine("Shared native 鲸 e\u0301 אבג", Font{Size: 24}, 0)
	want, wantOffset := layout.Rasterize(RGB(255, 255, 255), 1.5)
	if want == nil {
		t.Fatal("native baseline has no glyph ink")
	}
	reference := textRasterNativeReference(t, layout, RGB(255, 255, 255), 1.5)
	system := text.Shared()
	mask, color := system.MaskAtlas, system.ColorAtlas
	maskVersion, colorVersion := mask.Version(), color.Version()
	maskPixels, colorPixels := bytes.Clone(mask.Pix), bytes.Clone(color.Pix)
	var workers sync.WaitGroup
	start := make(chan struct{})
	errors := make(chan string, 4)
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for range 8 {
				img, offset := layout.Rasterize(RGB(255, 255, 255), 1.5)
				if img == nil || offset != wantOffset || !bytes.Equal(img.Pix, want.Pix) {
					errors <- "concurrent raster changed native ink or offset"
					return
				}
			}
		}()
	}
	close(start)
	for range 8 {
		if got := textRasterNativeReference(t, layout, RGB(255, 255, 255), 1.5); !bytes.Equal(got.Pix, reference.Pix) {
			t.Error("background raster corrupted the normal shared-atlas Painter")
		}
	}
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("concurrent layout/shared-engine lock order deadlocked")
	}
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	if mask != system.MaskAtlas || color != system.ColorAtlas || maskVersion != mask.Version() || colorVersion != color.Version() || !bytes.Equal(maskPixels, mask.Pix) || !bytes.Equal(colorPixels, color.Pix) {
		t.Fatal("private Rasterize calls wrote into or repacked the shared glyph atlases")
	}
	// The result owns its pixels; mutating it cannot poison a later raster.
	clear(want.Pix)
	if again, _ := layout.Rasterize(RGB(255, 255, 255), 1.5); again == nil || textRasterInk(again).Empty() {
		t.Fatal("returned bitmap aliases retained raster resources")
	}
}

func TestTesterSetTimeKeepsIndependentPainterClocks(t *testing.T) {
	var firstNow, secondNow time.Time
	first := NewTester(func(c *Context) { Box(c).Fill().Draw(func(p *Painter, _ Rect) { firstNow = p.Now() }) }, 80, 40)
	second := NewTester(func(c *Context) { Box(c).Fill().Draw(func(p *Painter, _ Rect) { secondNow = p.Now() }) }, 80, 40)
	at := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	first.SetTime(at)
	second.SetTime(at.Add(250 * time.Millisecond))
	first.Frame()
	second.Frame()
	if !firstNow.Equal(at) || !secondNow.Equal(at.Add(250*time.Millisecond)) {
		t.Fatalf("frozen Painter clocks crossed testers or advanced: first=%v second=%v", firstNow, secondNow)
	}
	first.SetTime(at.Add(time.Second))
	second.Frame()
	if !firstNow.Equal(at.Add(time.Second)) || !secondNow.Equal(at.Add(250*time.Millisecond)) {
		t.Fatal("changing one tester's frame time changed another painter")
	}
}

func TestTextRasterPaintKeepsOnlyCurrentTextInAccessTree(t *testing.T) {
	const current, previous = "Current native model", "Previous model raster"
	oldPixels, _ := ShapeTextLine(previous, Font{Size: 20}, 0).Rasterize(RGB(255, 255, 255), 1)
	if oldPixels == nil {
		t.Fatal("old native text fixture produced no bitmap")
	}
	oldBitmap := NewBitmap(oldPixels)
	tester := coreNewTester(func(c *context) {
		coreText(c, current).FontSize(20).TextColor(Transparent).Draw(func(p *Painter, box Rect) {
			p.Image(oldBitmap, Rect{box.X, box.Y, float32(oldPixels.Rect.Dx()), float32(oldPixels.Rect.Dy())}, FillBox)
		})
	}, 320, 80)
	tester.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if tester.h.access == nil {
		t.Fatal("AccessibilityOn did not publish a native AccessTree")
	}
	currentCount := 0
	for _, node := range tester.h.access.Nodes {
		if node.Label == previous || node.Value == previous {
			t.Fatal("paint-only old text appeared in the native accessibility tree")
		}
		if node.Role == platform.RoleText && node.Label == current {
			currentCount++
			if node.Bounds.W <= 0 || node.Bounds.H <= 0 {
				t.Fatal("current transparent text lost its accessible geometry")
			}
		}
	}
	if currentCount != 1 {
		t.Fatalf("native AccessTree contains %d current labels, want exactly one", currentCount)
	}
}
