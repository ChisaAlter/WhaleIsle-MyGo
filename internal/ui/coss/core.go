package coss

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// State is the set of interaction states coss styles by (README "States").
type State uint16

const (
	StateHover State = 1 << iota
	StatePressed
	StateFocusVisible
	StateDisabled
	StateInvalid
	StateOpen
	StateChecked
	StateIndeterminate
	StateSelected
	StateHighlighted
	StateLoading
)

// Has reports whether every bit of q is set.
func (s State) Has(q State) bool { return s&q == q }

// forced is OR-ed into every state this package reads, so the component
// gallery can render hover, pressed and focus without a pointer. Only
// tests set it.
var forced State

// segForce is forced for the second item of a segmented control only, as
// the gallery's reference forces one item. Only tests set it.
var segForce State

// interact reads e's pointer and focus state. Disabled controls take no
// hover, press or focus.
func interact(e ui.Element, disabled bool) State {
	s := forced
	if disabled || s.Has(StateDisabled) {
		return (s | StateDisabled) &^ (StateHover | StatePressed | StateFocusVisible)
	}
	if e.Hovered() {
		s |= StateHover
	}
	if e.Pressed() {
		s |= StatePressed | StateHover
	}
	if e.FocusVisible() {
		s |= StateFocusVisible
	}
	return s
}

// interactField is interact for text controls, whose :focus-visible holds
// on any focus, pointer or keyboard.
func interactField(e ui.Element, disabled bool) State {
	s := interact(e, disabled)
	if !s.Has(StateDisabled) && (e.Focused() || e.FocusWithin()) {
		s |= StateFocusVisible
	}
	return s
}

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// tok returns the window's tokens.
func tok(c *ui.Context) *design.Tokens { return design.For(c) }

// alpha returns c with its alpha multiplied by a (Tailwind's /N modifier).
func alpha(c ui.Color, a float32) ui.Color { return c.Alpha(a) }

var (
	white = ui.Color{R: 255, G: 255, B: 255, A: 255}
	black = ui.Color{A: 255}
)

// shadowList is a box-shadow list without allocation: CSS order, first
// layer on top.
type shadowList struct {
	l [8]design.Shadow
	n int
}

func (s *shadowList) add(x, y, blur, spread float32, c ui.Color) {
	if c.A == 0 || s.n == len(s.l) {
		return
	}
	s.l[s.n] = design.Shadow{X: x, Y: y, Blur: blur, Spread: spread, Color: c}
	s.n++
}

func (s *shadowList) inset(x, y, blur, spread float32, c ui.Color) {
	if c.A == 0 || s.n == len(s.l) {
		return
	}
	s.l[s.n] = design.Shadow{X: x, Y: y, Blur: blur, Spread: spread, Color: c, Inset: true}
	s.n++
}

// scale adds a shadow-* scale layer re-tinted to c at alpha a (shadow-xs/5,
// shadow-primary/24).
func (s *shadowList) scale(sc design.Shadows, c ui.Color, a float32) {
	for _, l := range sc {
		s.add(l.X, l.Y, l.Blur, l.Spread, c.Alpha(a))
	}
}

// ring adds ring-2 ring-offset-1: a gap in offset, then the ring, at amt
// of their alpha (the 150 ms box-shadow transition).
func (s *shadowList) ring(t *design.Tokens, amt float32) {
	if amt <= 0 {
		return
	}
	s.add(0, 0, 0, design.RingOffset, t.Background.Alpha(amt))
	s.add(0, 0, 0, design.RingOffset+design.RingWidth, t.Ring.Alpha(amt))
}

// apply sets the outer layers on e, bottom layer first as the toolkit paints
// in order.
func (s *shadowList) apply(e ui.Element) {
	for i := s.n - 1; i >= 0; i-- {
		if l := s.l[i]; !l.Inset {
			e.Shadow(l.X, l.Y, l.Blur, l.Spread, l.Color)
		}
	}
}

func (s *shadowList) hasInset() bool {
	for i := 0; i < s.n; i++ {
		if s.l[i].Inset {
			return true
		}
	}
	return false
}

// deco paints what CSS draws above a box's background and below its
// content: inset shadows (inside the padding box), the ::before overlay's
// 1 DIP highlight shadow (radius outer − 1), and any paint-only focus ring
// animation so focus transitions never trigger full-tree rebuilds.
type deco struct {
	insets  shadowList
	border  float32
	radius  float32
	before  bool
	beforeY float32
	beforeC ui.Color
	// fill paints an extra fill over the padding box (autofill tints,
	// pressed washes layered over a base).
	fill ui.Color
	// ring holds paint-only focus ring state when active or fading out.
	ring      *focusRingState
	ringColor ui.Color
	ringBg    ui.Color
	ringForce float32
}

type focusRingState struct {
	init   bool
	target bool
	from   float32
	start  time.Time
}

func (f *focusRingState) update(now time.Time, on bool) {
	if !f.init {
		f.init = true
		f.target = on
		f.from = b2f(on)
		f.start = time.Time{}
		return
	}
	if f.target != on {
		f.from = f.eval(now)
		f.target = on
		f.start = now
	}
}

func (f *focusRingState) eval(now time.Time) float32 {
	to := b2f(f.target)
	if f.start.IsZero() {
		return to
	}
	elapsed := now.Sub(f.start)
	if elapsed <= 0 {
		return f.from
	}
	if elapsed >= design.DurationDefault {
		return to
	}
	p := float32(elapsed) / float32(design.DurationDefault)
	eased := design.EaseInOut(p)
	return f.from + (to-f.from)*eased
}

func (f *focusRingState) animating(now time.Time) bool {
	if f.start.IsZero() {
		return false
	}
	return now.Sub(f.start) < design.DurationDefault
}

func (d *deco) setRing(c *ui.Context, e ui.Element, t *design.Tokens, on bool, ringColor ui.Color) {
	if ringColor.A == 0 {
		ringColor = t.Ring
	}
	d.ringColor = ringColor
	d.ringBg = t.Background
	if forced != 0 {
		d.ringForce = b2f(on)
		return
	}
	now := c.Now()
	st := ui.Local(e, "coss.ring.paint", func() focusRingState { return focusRingState{} })
	st.update(now, on)
	if on || st.eval(now) > 0 || st.animating(now) {
		d.ring = st
	}
}

func (d *deco) paint(p *ui.Painter, r ui.Rect) {
	if d.ring != nil || d.ringForce > 0 {
		amt := d.ringForce
		if d.ring != nil {
			now := p.Now()
			amt = d.ring.eval(now)
			if d.ring.animating(now) {
				p.After(16 * time.Millisecond)
			}
		}
		if amt > 0 {
			p.Shadow(r, d.radius, 0, 0, 0, design.RingOffset+design.RingWidth, d.ringColor.Alpha(amt))
			p.Shadow(r, d.radius, 0, 0, 0, design.RingOffset, d.ringBg.Alpha(amt))
		}
	}
	pad := ui.Rect{X: r.X + d.border, Y: r.Y + d.border, W: r.W - 2*d.border, H: r.H - 2*d.border}
	inner := max(d.radius-d.border, 0)
	if d.fill.A > 0 {
		p.Fill(pad, d.fill, inner)
	}
	for i := d.insets.n - 1; i >= 0; i-- {
		l := d.insets.l[i]
		if !l.Inset {
			continue
		}
		// An inset shadow offset by (x, y) is the padding box minus itself
		// moved by (x, y): the outer shadow of the moved box cast back.
		cast := ui.Rect{X: pad.X + l.X, Y: pad.Y + l.Y, W: pad.W, H: pad.H}
		p.Clip(pad, inner, func() {
			p.Shadow(cast, inner, -l.X, -l.Y, l.Blur, l.Spread, l.Color)
		})
	}
	if d.before && d.beforeC.A > 0 {
		p.Shadow(pad, inner, 0, d.beforeY, 0, 0, d.beforeC)
	}
}

func (d *deco) active() bool {
	return d.ring != nil || d.ringForce > 0 || d.fill.A > 0 || d.insets.hasInset() || d.before && d.beforeC.A > 0
}

// highlight sets the ::before shadow: 0 1px black/4% in light, 0 -1px
// white/6% in dark (or darkA when given).
func (d *deco) highlight(t *design.Tokens, lightA, darkA float32) {
	d.before = true
	if t.IsDark() {
		d.beforeY, d.beforeC = -1, white.Alpha(darkA)
	} else {
		d.beforeY, d.beforeC = 1, black.Alpha(lightA)
	}
}

// finish applies the shadows and decoration to e.
func finish(e ui.Element, sh *shadowList, d *deco) {
	sh.apply(e)
	if d != nil {
		d.insets = *sh
		if d.active() {
			dd := *d
			e.Draw(dd.paint)
		}
	}
}

// text builds a single-line text in a coss type pair.
func text(c *ui.Context, s string, ts design.TextStyle, weight int, col ui.Color) ui.Element {
	return ui.Text(c, s).FontSize(ts.Size).FixedLineHeight(ts.LineHeight).FontWeight(weight).TextColor(col).SingleLine()
}

// para builds wrapping text in a coss type pair.
func para(c *ui.Context, s string, ts design.TextStyle, weight int, col ui.Color) ui.Element {
	return ui.Text(c, s).FontSize(ts.Size).FixedLineHeight(ts.LineHeight).FontWeight(weight).TextColor(col)
}

// ringAmount returns the static ring opacity without scheduling a rebuild
// animation; interactive controls paint animated focus rings via deco.setRing.
func ringAmount(_ ui.Element, on bool) float32 {
	return b2f(on)
}

// isDisabled folds the gallery's forced disabled state into a component's
// own flag.
func isDisabled(d bool) bool { return d || forced.Has(StateDisabled) }

// disabledLook dims a disabled control (opacity-64). It also marks the
// element disabled for input and assistive technology, replacing the
// toolkit's own .48 dim with coss's 64%.
func disabledLook(e ui.Element, s State) {
	if s.Has(StateDisabled) {
		e.Disabled(true).DisabledOpacity(design.OpacityDisabled)
	}
}

// border sets a 1 DIP border, transparent where it would only repeat the
// fill (under a disabled control's opacity the toolkit paints each layer
// translucent, and a same-color border would show darker than the fill).
func border(e ui.Element, b, fill ui.Color) {
	if b == fill {
		b = ui.Transparent
	}
	e.Border(design.BorderWidth, b)
}

// withKey gives the next element, a component's outermost, its key.
func withKey(c *ui.Context, key any) {
	if key != nil {
		c.Key(key)
	}
}
