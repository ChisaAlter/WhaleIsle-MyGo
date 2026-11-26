package coss

import (
	"image"
	"image/draw"
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// FlipTextOptions configures a 400ms vertical flip-text control
// (motion.css: `perspective: 140px`, `rotateX(80deg)` -> `0` / `-80deg`).
type FlipTextOptions struct {
	Key        string
	Color      ui.Color
	Font       string
	FontSize   float32
	LineHeight float32
	FontWeight int
}

type flipTimeline struct {
	Shown, Outgoing  string
	Initialized      bool
	HasOutgoing      bool
	Reduced          bool
	Started, Expires time.Time
}

func (s *flipTimeline) update(text string, now time.Time, reduced bool) {
	if s.HasOutgoing && !now.Before(s.Expires) {
		s.HasOutgoing = false
	}
	if s.Reduced && !reduced && s.HasOutgoing {
		s.Started = now
	}
	s.Reduced = reduced
	if !s.Initialized || now.IsZero() {
		s.Shown, s.Initialized = text, true
		if now.IsZero() {
			s.HasOutgoing = false
		}
		return
	}
	if text == s.Shown {
		return
	}
	previous := s.Shown
	s.Shown = text
	if reduced {
		s.HasOutgoing = false
		return
	}
	fresh := s.recipeOver(now)
	if fresh {
		s.Started = now
	}
	if fresh || s.Outgoing != previous {
		s.Expires = now.Add(design.MotionFlip)
	}
	s.Outgoing, s.HasOutgoing = previous, true
}

func (s *flipTimeline) recipeOver(now time.Time) bool {
	if !s.HasOutgoing || now.IsZero() {
		return true
	}
	return !now.Before(s.Started.Add(design.MotionFlip))
}

type flipInk struct {
	text   string
	font   ui.Font
	color  ui.Color
	scale  float32
	w, h   float32
	layout *ui.TextLayout
	pixels *image.RGBA
}

func (ink *flipInk) prepare(text string, font ui.Font, color ui.Color, scale, w, h float32) {
	if ink.layout != nil && ink.text == text && ink.font == font && ink.color == color && ink.scale == scale && ink.w == w && ink.h == h {
		return
	}
	*ink = flipInk{text: text, font: font, color: color, scale: scale, w: w, h: h}
	ink.layout = ui.ShapeTextLine(text, font, w)
	ink.pixels = image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(w*scale))), int(math.Ceil(float64(h*scale)))))
	pixels, offset := ink.layout.Rasterize(color, scale)
	if pixels == nil {
		return
	}
	_, naturalHeight := ink.layout.Size()
	origin := image.Pt(int(math.Round(float64(offset.X*scale))), int(math.Round(float64((offset.Y+(h-naturalHeight)/2)*scale))))
	draw.Draw(ink.pixels, pixels.Bounds().Add(origin), pixels, pixels.Bounds().Min, draw.Src)
}

type flipTextState struct {
	Timeline          flipTimeline
	Current, Outgoing flipInk
	CurrentRaster     ui.DecorativeRaster
	OutgoingRaster    ui.DecorativeRaster
}

// FlipText paints a single-line label that flips vertically along rotateX over
// design.MotionFlip (400ms) on design.EaseInOut when its value changes. When
// settled or under Preferences.ReduceMotion it returns a plain ui.Text with 0 Hz
// idle overhead.
func FlipText(c *ui.Context, value string, opts FlipTextOptions) ui.Element {
	family := opts.Font
	if family == "" {
		family = c.Theme().Font
	}
	size := opts.FontSize
	if size <= 0 {
		size = 14
	}
	weight := opts.FontWeight
	if weight <= 0 {
		weight = 400
	}
	color := opts.Color
	if color.A == 0 {
		color = c.Theme().Text
	}
	font := ui.Font{Family: family, Size: size, Weight: weight}
	ctx := c
	if opts.Key != "" {
		ctx = c.Key(opts.Key)
	}
	label := ui.Text(ctx, value).MinWidth(0).SingleLine().Ellipsis("…").
		Font(font.Family).FontSize(font.Size).FontWeight(font.Weight).TextColor(color)
	if opts.LineHeight > 0 {
		label.FixedLineHeight(opts.LineHeight)
	}
	state := ui.Local(label, "flip-text", func() flipTextState { return flipTextState{} })
	reduced := c.Preferences().ReduceMotion
	now := c.Now()
	state.Timeline.update(value, now, reduced)
	if reduced || state.Timeline.recipeOver(now) {
		state.Current, state.Outgoing = flipInk{}, flipInk{}
		return label
	}
	label.TextColor(ui.Transparent).DrawOver(func(p *ui.Painter, box ui.Rect) {
		if box.W <= 0 || box.H <= 0 {
			return
		}
		p.Clip(box, 0, func() {
			paintNow := p.Now()
			elapsed := paintNow.Sub(state.Timeline.Started)
			state.Current.prepare(value, font, color, p.Scale(), box.W, box.H)
			state.Outgoing.prepare(state.Timeline.Outgoing, font, color, p.Scale(), box.W, box.H)
			if elapsed >= design.MotionFlip {
				_, height := state.Current.layout.Size()
				p.TextLayout(state.Current.layout, box.X, box.Y+(box.H-height)/2, color)
				return
			}
			progress := float64(design.EaseInOut(float32(elapsed) / float32(design.MotionFlip)))
			paint := func(ink *flipInk, raster *ui.DecorativeRaster, angle, opacity float64) {
				pixels, offset := projectFlip(ink.pixels, box.W, box.H, p.Scale(), angle, opacity)
				if pixels == nil {
					return
				}
				raster.Paint(p, pixels, ui.Rect{X: box.X + offset.X, Y: box.Y + offset.Y, W: float32(pixels.Bounds().Dx()) / p.Scale(), H: float32(pixels.Bounds().Dy()) / p.Scale()})
			}
			paint(&state.Outgoing, &state.OutgoingRaster, -80*progress, 1-progress)
			paint(&state.Current, &state.CurrentRaster, 80*(1-progress), progress)
			if remaining := design.MotionFlip - elapsed; remaining > 0 {
				p.After(remaining)
			}
			p.AnimationFrame()
		})
	})
	return label
}

func projectFlip(src *image.RGBA, seatWidth, seatHeight, scale float32, rotateXDeg, opacity float64) (*image.RGBA, ui.Point) {
	if src == nil || src.Bounds().Empty() || seatWidth <= 0 || seatHeight <= 0 || scale <= 0 || opacity <= 0 {
		return nil, ui.Point{}
	}
	sine, cosine := math.Sincos(rotateXDeg * math.Pi / 180)
	const perspective = 140.0
	pixelsPerDIP := float64(scale)
	bounds := src.Bounds()
	width, height := float64(seatWidth), float64(seatHeight)
	centerX, centerY := width/2, height/2
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, corner := range [4][2]float64{{0, 0}, {width, 0}, {0, height}, {width, height}} {
		x, y := corner[0]-centerX, corner[1]-centerY
		denominator := 1 - y*sine/perspective
		projectedX, projectedY := centerX+x/denominator, centerY+y*cosine/denominator
		minX, minY = min(minX, projectedX), min(minY, projectedY)
		maxX, maxY = max(maxX, projectedX), max(maxY, projectedY)
	}
	left, top := int(math.Floor(minX*pixelsPerDIP)), int(math.Floor(minY*pixelsPerDIP))
	right, bottom := int(math.Ceil(maxX*pixelsPerDIP)), int(math.Ceil(maxY*pixelsPerDIP))
	dstW, dstH := right-left, bottom-top
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	srcW, srcH := bounds.Dx(), bounds.Dy()
	srcMinX, srcMinY := bounds.Min.X, bounds.Min.Y
	srcPix, srcStride := src.Pix, src.Stride
	dstPix, dstStride := dst.Pix, dst.Stride
	baseOff := src.PixOffset(srcMinX, srcMinY)

	if rotateXDeg == 0 && opacity == 1 && left == 0 && top == 0 && dstW == srcW && dstH == srcH {
		for y := 0; y < dstH; y++ {
			so := baseOff + y*srcStride
			do := y * dstStride
			copy(dstPix[do:do+dstW*4], srcPix[so:so+srcW*4])
		}
		return dst, ui.Point{}
	}

	op := float32(opacity)
	invScale := 1.0 / pixelsPerDIP
	for y := 0; y < dstH; y++ {
		projectedY := (float64(top+y)+.5)*invScale - centerY
		denominator := cosine + projectedY*sine/perspective
		sourceY := projectedY / denominator
		perspectiveDivisor := 1 - sourceY*sine/perspective
		sampleY := (centerY+sourceY)*pixelsPerDIP - .5
		y0 := int(math.Floor(sampleY))
		y1 := y0 + 1
		if y1 < 0 || y0 >= srcH {
			continue
		}
		fy := float32(sampleY - float64(y0))
		fy1 := 1 - fy
		y0In := y0 >= 0 && y0 < srcH
		y1In := y1 >= 0 && y1 < srcH
		row0 := baseOff + y0*srcStride
		row1 := baseOff + y1*srcStride
		dstRow := y * dstStride
		stepX := perspectiveDivisor
		sampleX := (centerX+(float64(left)+.5)*invScale*perspectiveDivisor-centerX*perspectiveDivisor)*pixelsPerDIP - .5
		for x := 0; x < dstW; x++ {
			x0 := int(math.Floor(sampleX))
			x1 := x0 + 1
			fx := float32(sampleX - float64(x0))
			sampleX += stepX
			if x1 < 0 || x0 >= srcW {
				continue
			}
			fx1 := 1 - fx
			x0In := x0 >= 0 && x0 < srcW
			x1In := x1 >= 0 && x1 < srcW

			var a0, a1, a2, a3 float32
			var b0, b1, b2, b3 float32
			var c0, c1, c2, c3 float32
			var d0, d1, d2, d3 float32
			if y0In {
				if x0In {
					o := row0 + x0*4
					a0, a1, a2, a3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
				if x1In {
					o := row0 + x1*4
					b0, b1, b2, b3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
			}
			if y1In {
				if x0In {
					o := row1 + x0*4
					c0, c1, c2, c3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
				if x1In {
					o := row1 + x1*4
					d0, d1, d2, d3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
			}
			if a3 == 0 && b3 == 0 && c3 == 0 && d3 == 0 {
				continue
			}
			do := dstRow + x*4
			dstPix[do] = uint8(((a0*fx1+b0*fx)*fy1+(c0*fx1+d0*fx)*fy)*op + 0.5)
			dstPix[do+1] = uint8(((a1*fx1+b1*fx)*fy1+(c1*fx1+d1*fx)*fy)*op + 0.5)
			dstPix[do+2] = uint8(((a2*fx1+b2*fx)*fy1+(c2*fx1+d2*fx)*fy)*op + 0.5)
			dstPix[do+3] = uint8(((a3*fx1+b3*fx)*fy1+(c3*fx1+d3*fx)*fy)*op + 0.5)
		}
	}
	return dst, ui.Point{X: float32(float64(left) / pixelsPerDIP), Y: float32(float64(top) / pixelsPerDIP)}
}

func selectPopoverReveal(c *ui.Context, t *design.Tokens, panel ui.Element) ui.Element {
	if c == nil || c.Preferences().ReduceMotion || forced != 0 {
		return panel
	}
	now := c.Now()
	if now.IsZero() {
		return panel
	}
	start := ui.Local(panel, "coss.select.reveal", func() time.Time { return now })
	if start.IsZero() || start.After(now) {
		*start = now
	}
	elapsed := now.Sub(*start)
	if elapsed >= design.MotionPopover {
		return panel
	}
	c.AnimationFrame()
	p := design.EaseInOut(max(0, float32(elapsed)/float32(design.MotionPopover)))
	if t != nil && t.Glass {
		return panel.OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: (1 - p) * 4} })
	}
	return panel.Opacity(p)
}
