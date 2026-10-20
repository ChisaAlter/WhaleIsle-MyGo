package widgets

import (
	"github.com/egoist/mygo/ui"
)

// TextShimmerOptions describes a TextShimmer.
type TextShimmerOptions struct {
	// Text is the label to show. It is one line: text too long for the room
	// it gets ends with an ellipsis, as a single-line label does.
	Text string
	// Active sweeps a highlight across the text while work runs. Inactive
	// text stands still and requests no frames.
	Active bool
	// Size is the text's size in DIPs, 13 when 0.
	Size float32
	// Weight is the text's weight from 100 to 900, 400 when 0.
	Weight int
	// Color is the text's color, the theme's muted text when zero.
	Color ui.Color
}

// shimmerPeriod and shimmerDelay time the sweep: a highlight crosses the text
// once every period, after standing still for the delay, as the design
// system's TextShimmer does.
const (
	shimmerPeriod = 1500
	shimmerDelay  = 300
)

// TextShimmer shows a label, sweeping a highlight across it while Active for
// work that is still running:
//
//	widgets.TextShimmer(c, widgets.TextShimmerOptions{Text: "Thinking", Active: running})
//
// The text is painted by the primitive itself, so the highlight and the text
// it crosses cannot come apart. Long text ends with an ellipsis, and the
// element asks for no frames unless it is active. Without motion — the
// desktop's reduce-motion setting, or Active false — the text is simply
// static.
func TextShimmer(c *ui.Context, o TextShimmerOptions) ui.Element {
	t := c.Theme()
	size := o.Size
	if size <= 0 {
		size = 13
	}
	color := o.Color
	if color.A == 0 {
		color = t.TextMuted
	}
	span := ui.Span{Text: o.Text, Size: size, Weight: o.Weight, Color: color}
	w, h := c.MeasureText(0, span)
	if h <= 0 {
		h = size * 1.35
	}
	shimmer := ui.Box(c).Width(w).Height(h).MinWidth(0)
	if o.Text != "" {
		// The label is painted, so name it here for assistive technology and
		// for finding it in tests.
		shimmer.Label(o.Text)
	}
	active := o.Active && o.Text != "" && !c.Preferences().ReduceMotion
	highlight := shimmerHighlight(t, color)
	phase := float32(0)
	if active {
		c.AnimationFrame()
		ms := c.Now().UnixMilli() % shimmerPeriod
		phase = float32(ms-shimmerDelay) / float32(shimmerPeriod-shimmerDelay)
	}
	full := w
	shimmer.Draw(func(p *ui.Painter, r ui.Rect) {
		text := o.Text
		if full > r.W {
			// Only text that does not fit costs a search for the room left.
			text = fitSpan(p, span, r.W)
		}
		if text == "" {
			return
		}
		p.RichText(r.X, r.Y, 0, withSpanText(span, text, color))
		if !active || phase <= 0 || phase >= 1 {
			return
		}
		band := max(r.W*0.45, 28)
		x := r.X - band + (r.W+band)*phase
		p.Clip(ui.Rect{X: x, Y: r.Y, W: band, H: r.H}, 0, func() {
			p.RichText(r.X, r.Y, 0, withSpanText(span, text, highlight))
		})
	})
	return shimmer
}

// shimmerHighlight returns the color the sweep paints: the text color taken
// toward the theme's primary text, or, for text that is already primary,
// toward the background, so that the highlight always reads against both the
// text and what is behind it.
func shimmerHighlight(t *ui.Theme, color ui.Color) ui.Color {
	if color == t.Text {
		return color.Mix(t.Background, 0.45)
	}
	return color.Mix(t.Text, 0.6)
}

// withSpanText returns span showing text in color.
func withSpanText(span ui.Span, text string, color ui.Color) ui.Span {
	span.Text, span.Color = text, color
	return span
}

// fitSpan returns the longest prefix of span's text that fits in width with an
// ellipsis, as a single-line label cuts its text, or the whole text when it
// fits. It returns "" for no room at all.
func fitSpan(p *ui.Painter, span ui.Span, width float32) string {
	if width <= 0 || span.Text == "" {
		return ""
	}
	if w, _ := p.MeasureText(0, withSpanText(span, span.Text, span.Color)); w <= width {
		return span.Text
	}
	runes := []rune(span.Text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		candidate := string(runes[:mid]) + "…"
		if w, _ := p.MeasureText(0, withSpanText(span, candidate, span.Color)); w <= width {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return string(runes[:lo]) + "…"
}
