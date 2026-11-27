package chat

import (
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
)

func chatTokens(c *ui.Context) *design.Tokens {
	return design.For(c)
}

func chatCardSurface(c *ui.Context, e ui.Element, radius float32) ui.Element {
	t := design.For(c)
	e.Radius(radius).TextColor(t.CardForeground)
	if t.Glass || chrome.BackdropActive() {
		dark := t.IsDark()
		tint := t.Card
		if !t.Glass && chrome.BackdropActive() {
			tint = chrome.MaterialFill(t.Card, chrome.MaterialAlpha(chrome.MaterialPanel, chrome.BackdropPreferences(), dark))
		}
		e.Background(tint).Border(design.BorderWidth*0.5, t.Border)
		if dark {
			e.Shadow(0, 2, 6, 0, design.TW.Black.Alpha(0.12)).
				Shadow(0, 8, 20, -2, design.TW.Black.Alpha(0.22))
		} else {
			e.Shadow(0, 1, 4, 0, design.TW.Black.Alpha(0.04)).
				Shadow(0, 6, 16, -2, design.TW.Black.Alpha(0.07))
		}
		return e
	}
	return e.Border(design.BorderWidth, t.Border).Background(t.Card)
}

func chatInsetSurface(c *ui.Context, e ui.Element, radius float32) ui.Element {
	t := design.For(c)
	return e.Background(t.Muted).Border(design.BorderWidth, t.Border).Radius(radius)
}

func chatPopupSurface(c *ui.Context, e ui.Element, radius float32) ui.Element {
	return chrome.ApplyMenuSurface(e, design.For(c), radius)
}

func chatStateColor(t *design.Tokens, s rowStatus) ui.Color {
	switch s {
	case statusRunning:
		return t.MutedForeground
	case statusDone:
		return t.Success
	case statusWarning:
		return t.Warning
	case statusError:
		return t.Destructive
	default:
		return t.MutedForeground.Alpha(0.8)
	}
}

func chatStateDot(c *ui.Context, s rowStatus) ui.Element {
	t := design.For(c)
	color := chatStateColor(t, s)
	const slot = 10
	const core = 6
	dot := ui.Box(c).Size(slot, slot).Shrink(0).Role(ui.RoleNone)
	pulse := float32(0.35)
	if s == statusRunning && !c.Preferences().ReduceMotion {
		pulse = dot.Loop("chat-state-dot", 1500*time.Millisecond, ui.Bounce(design.EaseInOut))
	}
	ring := 4.5 + 0.45*pulse
	alpha := 0.55 - 0.4*pulse
	dot.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		if s == statusRunning {
			p.Stroke(ui.Rect{X: cx - ring, Y: cy - ring, W: ring * 2, H: ring * 2}, color.Alpha(alpha), ring, 1.2)
		}
		p.Fill(ui.Rect{X: cx - core/2, Y: cy - core/2, W: core, H: core}, color, core/2)
	})
	return dot
}

type textShimmerOptions struct {
	Text   string
	Active bool
	Font   string
	Size   float32
	Weight int
	Color  ui.Color
}

const (
	chatShimmerPeriod = 1500
	chatShimmerDelay  = 300
)

func chatTextShimmer(c *ui.Context, o textShimmerOptions) ui.Element {
	t := design.For(c)
	size := o.Size
	if size <= 0 {
		size = 13
	}
	color := o.Color
	if color.A == 0 {
		color = t.MutedForeground
	}
	span := ui.Span{Text: o.Text, Font: o.Font, Size: size, Weight: o.Weight, Color: color}
	w, h := c.MeasureText(0, span)
	if h <= 0 {
		h = size * 1.35
	}
	shimmer := ui.Box(c).Width(w).Height(h).MinWidth(0)
	if o.Text != "" {
		shimmer.Label(o.Text)
	}
	active := o.Active && o.Text != "" && !c.Preferences().ReduceMotion
	highlight := color.Mix(t.Foreground, 0.6)
	if color == t.Foreground {
		highlight = color.Mix(t.Background, 0.45)
	}
	full := w
	shimmer.Draw(func(p *ui.Painter, r ui.Rect) {
		phase := float32(0)
		if active {
			p.AnimationFrame()
			ms := p.Now().UnixMilli() % chatShimmerPeriod
			phase = float32(ms-chatShimmerDelay) / float32(chatShimmerPeriod-chatShimmerDelay)
		}
		text := o.Text
		if full > r.W {
			text = fitShimmerSpan(p, span, r.W)
		}
		if text == "" {
			return
		}
		s := span
		s.Text, s.Color = text, color
		p.RichText(r.X, r.Y, 0, s)
		if !active || phase <= 0 || phase >= 1 {
			return
		}
		band := max(r.W*0.45, 28)
		x := r.X - band + (r.W+band)*phase
		p.Clip(ui.Rect{X: x, Y: r.Y, W: band, H: r.H}, 0, func() {
			hs := span
			hs.Text, hs.Color = text, highlight
			p.RichText(r.X, r.Y, 0, hs)
		})
	})
	return shimmer
}

func fitShimmerSpan(p *ui.Painter, span ui.Span, width float32) string {
	if width <= 0 || span.Text == "" {
		return ""
	}
	s := span
	if w, _ := p.MeasureText(0, s); w <= width {
		return span.Text
	}
	const ellipsis = "…"
	s.Text = ellipsis
	if ew, _ := p.MeasureText(0, s); ew > width {
		return ""
	}
	var cuts []int
	for i := range span.Text {
		cuts = append(cuts, i)
	}
	cuts = append(cuts, len(span.Text))
	lo, hi := 0, len(cuts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		s.Text = span.Text[:cuts[mid]] + ellipsis
		if w, _ := p.MeasureText(0, s); w <= width {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 && utf8.RuneCountInString(span.Text) > 0 {
		return ellipsis
	}
	return span.Text[:cuts[lo]] + ellipsis
}

type chatAckSeen struct {
	text  string
	until time.Time
}

func chatAckLineKeyed(c *ui.Context, id, key, label string) {
	seat := ui.Box(c.Key("ack/"+id)).Absolute().Size(0, 0).PassThrough()
	st := ui.Local(seat, "state", func() chatAckSeen { return chatAckSeen{} })
	now := time.Now()
	if st.text != key {
		st.text = key
		if key != "" {
			st.until = now.Add(6 * time.Second)
		}
	}
	if key == "" || !now.Before(st.until) {
		return
	}
	c.After(st.until.Sub(now))
	t := design.For(c)
	ui.Text(c, label).FillWidth().FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.MutedForeground)
}
