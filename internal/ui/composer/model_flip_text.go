package composer

import (
	"image"
	"image/draw"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// ModelSelect's FlipText keeps its CSS animation mounted across rapid text
// changes. Its outgoing-value cleanup effect has a separate 400ms clock.
const modelFlipDuration = 400 * time.Millisecond

type modelFlipTimeline struct {
	Shown, Outgoing  string
	Initialized      bool
	HasOutgoing      bool
	Reduced          bool
	Started, Expires time.Time
}

func (s *modelFlipTimeline) update(text string, now time.Time, reduced bool) {
	if s.HasOutgoing && !now.Before(s.Expires) {
		s.HasOutgoing = false
	}
	if s.Reduced && !reduced && s.HasOutgoing {
		// Removing CSS animation:none restarts the still-mounted recipe.
		s.Started = now
	}
	s.Reduced = reduced
	if !s.Initialized {
		s.Shown, s.Initialized = text, true
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
	if !s.HasOutgoing {
		s.Started = now
	}
	if !s.HasOutgoing || s.Outgoing != previous {
		s.Expires = now.Add(modelFlipDuration)
	}
	s.Outgoing, s.HasOutgoing = previous, true
}

type modelFlipInk struct {
	text   string
	font   ui.Font
	color  ui.Color
	scale  float32
	w, h   float32
	layout *ui.TextLayout
	pixels *image.RGBA
}

func (ink *modelFlipInk) prepare(text string, font ui.Font, color ui.Color, scale, w, h float32) {
	if ink.layout != nil && ink.text == text && ink.font == font && ink.color == color && ink.scale == scale && ink.w == w && ink.h == h {
		return
	}
	*ink = modelFlipInk{text: text, font: font, color: color, scale: scale, w: w, h: h}
	ink.layout = ui.ShapeTextLine(text, font, w)
	ink.pixels = image.NewRGBA(image.Rect(0, 0, int(math.Ceil(float64(w*scale))), int(math.Ceil(float64(h*scale)))))
	pixels, offset := ink.layout.Rasterize(color, scale)
	if pixels == nil {
		return
	}
	_, naturalHeight := ink.layout.Size()
	origin := image.Pt(int(math.Round(float64(offset.X*scale))), int(math.Round(float64((offset.Y+(h-naturalHeight)/2)*scale))))
	// Both spans are overflow:hidden/ellipsis within the NEW text's seat,
	// before rotation. The outgoing value never determines the layout width.
	draw.Draw(ink.pixels, pixels.Bounds().Add(origin), pixels, pixels.Bounds().Min, draw.Src)
}

type modelFlipState struct {
	Timeline          modelFlipTimeline
	Current, Outgoing modelFlipInk
	CurrentRaster     ui.DecorativeRaster
	OutgoingRaster    ui.DecorativeRaster
}

// The ordinary Text node owns current-value layout and accessibility. Only
// its ink is replaced during the fixed ModelSelect animation; outgoing ink
// is never an accessible element and cannot widen the trigger or its badge.
func modelFlipText(c *ui.Context, key, value string, color ui.Color, fontSize float32) ui.Element {
	font := ui.Font{Family: c.Theme().Font, Size: fontSize, Weight: 400}
	label := ui.Text(c.Key(key), value).MinWidth(0).SingleLine().FixedLineHeight(20).
		Font(font.Family).FontSize(font.Size).FontWeight(font.Weight).TextColor(color)
	state := ui.Local(label, "model-flip", func() modelFlipState { return modelFlipState{} })
	reduced := c.Preferences().ReduceMotion
	state.Timeline.update(value, c.Now(), reduced)
	if !state.Timeline.HasOutgoing || reduced {
		state.Current, state.Outgoing = modelFlipInk{}, modelFlipInk{}
		return label
	}
	label.TextColor(ui.Transparent).DrawOver(func(p *ui.Painter, box ui.Rect) {
		if box.W <= 0 || box.H <= 0 {
			return
		}
		// The original name/effort wrapper clips the transformed recipe. Keep
		// the projection's near-edge expansion inside that wrapper, away from
		// the independent release badge and neighbouring trigger controls.
		p.Clip(box, 0, func() {
			now := p.Now()
			elapsed := now.Sub(state.Timeline.Started)
			state.Current.prepare(value, font, color, p.Scale(), box.W, box.H)
			if elapsed >= modelFlipDuration {
				_, height := state.Current.layout.Size()
				p.TextLayout(state.Current.layout, box.X, box.Y+(box.H-height)/2, color)
				if now.Before(state.Timeline.Expires) {
					p.After(state.Timeline.Expires.Sub(now))
				} else {
					state.Timeline.HasOutgoing = false
					state.Outgoing = modelFlipInk{}
				}
				return
			}
			state.Outgoing.prepare(state.Timeline.Outgoing, font, color, p.Scale(), box.W, box.H)
			progress := cssBezier(float64(elapsed)/float64(modelFlipDuration), .4, 0, .2, 1)
			paint := func(ink *modelFlipInk, raster *ui.DecorativeRaster, angle, opacity float64) {
				pixels, offset := projectModelFlip(ink.pixels, box.W, box.H, p.Scale(), angle, opacity)
				if pixels == nil {
					return
				}
				raster.Paint(p, pixels, ui.Rect{X: box.X + offset.X, Y: box.Y + offset.Y, W: float32(pixels.Bounds().Dx()) / p.Scale(), H: float32(pixels.Bounds().Dy()) / p.Scale()})
			}
			paint(&state.Outgoing, &state.OutgoingRaster, -80*progress, 1-progress)
			paint(&state.Current, &state.CurrentRaster, 80*(1-progress), progress)
			p.AnimationFrame()
		})
	})
	return label
}
