package ui

import (
	"image"
	"math"

	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
)

// ShapeTextLine shapes one display line using the same native ellipsis and
// no-wrap policy as Text.SingleLine. A positive width is the available width
// in DIPs; zero uses the line's natural width. Fonts, fallback, bidi and
// grapheme-safe truncation are owned by the system text engine.
func ShapeTextLine(s string, font Font, width float32) *TextLayout {
	l := text.Shared().Shape(text.Params{Text: s, Style: font.style(), Width: width, MaxLines: 1, NoWrap: true})
	return retainTextLayout(l, nil)
}

// Rasterize paints retained shaped text into an owned, transparent,
// premultiplied sRGB RGBA image at scale pixels per DIP. It uses native glyph
// rasterization with grayscale coverage (never screen-subpixel coverage),
// retaining rich span colors, backgrounds, decorations and native color glyphs.
//
// The image starts at pixel (0,0). The returned Point places its top-left
// relative to the layout origin in DIPs, including ink overhang and a one-pixel
// antialiasing margin. Draw it there at image size divided by scale; Size still
// gives the layout's logical extent. No view, window or Tester is created.
//
// Rasterize is safe from any goroutine, including alongside Painter.TextLayout.
// It returns nil for a nil layout, nonpositive/nonfinite scale, no paintable
// geometry, unrepresentable image bounds, or glyphs exceeding atlas capacity.
func (t *TextLayout) Rasterize(color Color, scale float32) (*image.RGBA, Point) {
	if t == nil || scale <= 0 || math.IsNaN(float64(scale)) || math.IsInf(float64(scale), 0) {
		return nil, Point{}
	}
	var display scene.Scene
	ok := false
	func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.l == nil {
			return
		}
		text.Shared().WithRasterSystem(func(system *text.System) {
			// Only this temporary engine's text/glyph-run fields are used. The
			// native text engine is borrowed under its lock, while these atlases
			// and scene are independent of every window and other raster call.
			runtime := &engine{text: system}
			params := system.TextParams()
			for attempt := 0; attempt < 3; attempt++ {
				display.Reset(0, 0, scene.Color{})
				display.Scale, display.Text = scale, params
				display.MaskAtlas, display.ColorAtlas = system.MaskAtlas, system.ColorAtlas
				const extent = math.MaxFloat32 / 4
				painter := Painter{rt: runtime, s: &display, scale: scale, opacity: 1, clip: Rect{-extent, -extent, 2 * extent, 2 * extent}, opaque: false}
				painter.textLayout(t.l, 0, 0, color, textStyle{size: t.l.Params.Style.FontSize()}, newSpanPaint(t.spans))
				if !system.Full() {
					ok = true
					break
				}
				// Match the native frame's two atlas-room repaints. Never return
				// a silently incomplete bitmap if the private atlases still fill.
				if attempt < 2 {
					system.MakeRoom()
				}
			}
		})
	}()
	if !ok {
		return nil, Point{}
	}
	bounds, ok := textRasterBounds(&display)
	if !ok || bounds.Empty() {
		return nil, Point{}
	}
	offset := Point{float32(bounds.Min.X) / scale, float32(bounds.Min.Y) / scale}
	// Translate already-rasterized glyph positions by whole pixels. Repainting
	// at a new fractional DIP origin would change native hinting and coverage.
	dx, dy := float32(bounds.Min.X), float32(bounds.Min.Y)
	for i := range display.Glyphs {
		display.Glyphs[i].X -= dx
		display.Glyphs[i].Y -= dy
	}
	for i := range display.Ops {
		if display.Ops[i].Kind == scene.OpFill {
			display.Ops[i].Rect.X -= dx
			display.Ops[i].Rect.Y -= dy
		}
	}
	display.Width, display.Height = bounds.Dx(), bounds.Dy()
	pixels := raster.NewImage(display.Width, display.Height)
	raster.Render(pixels, &display)
	return &image.RGBA{Pix: pixels.RGBA(), Stride: 4 * display.Width, Rect: image.Rect(0, 0, display.Width, display.Height)}, offset
}

// Painter.textLayout emits only glyphs and rectangular span/decorative fills.
// Use their device-pixel bounds, as the CPU scene renderer does, rather than
// assuming that advance width and line height contain the font's ink.
func textRasterBounds(display *scene.Scene) (image.Rectangle, bool) {
	var bounds image.Rectangle
	valid := true
	add := func(x, y, w, h float32) {
		if w <= 0 || h <= 0 {
			return
		}
		coordinates := [4]float64{math.Floor(float64(x) - 1), math.Floor(float64(y) - 1), math.Ceil(float64(x) + float64(w) + 1), math.Ceil(float64(y) + float64(h) + 1)}
		limit := float64(int(^uint(0)>>1) / 4)
		for _, coordinate := range coordinates {
			if math.IsNaN(coordinate) || math.IsInf(coordinate, 0) || coordinate < -limit || coordinate > limit {
				valid = false
				return
			}
		}
		bounds = bounds.Union(image.Rect(int(coordinates[0]), int(coordinates[1]), int(coordinates[2]), int(coordinates[3])))
	}
	for _, glyph := range display.Glyphs {
		add(glyph.X, glyph.Y, glyph.W, glyph.H)
	}
	for _, op := range display.Ops {
		if op.Kind == scene.OpFill {
			add(op.Rect.X, op.Rect.Y, op.Rect.W, op.Rect.H)
		}
	}
	if !bounds.Empty() && bounds.Dx() > int(^uint(0)>>1)/4/bounds.Dy() {
		valid = false
	}
	return bounds, valid
}
