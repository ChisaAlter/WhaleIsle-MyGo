package design

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// TextStyle is a font size with its line height, in DIPs: a coss text-*
// pair. Fonts stay the app's CJK-capable system families; only the metrics
// come from coss.
type TextStyle struct {
	Size, LineHeight float32
}

// Shadow is one box-shadow layer. Inset layers paint inside the box.
type Shadow struct {
	X, Y, Blur, Spread float32
	Color              ui.Color
	Inset              bool
}

// Shadows is a box-shadow list, first layer on top as in CSS.
type Shadows []Shadow

// Spacing unit: Tailwind v4's --spacing, 4 DIPs. Sp(n) is the n-th step.
const Unit float32 = 4

// Sp returns n spacing steps in DIPs (p-2 → Sp(2) = 8).
func Sp(n float32) float32 { return n * Unit }

// Focus ring recipes (tokens.json focusRing).
const (
	// RingWidth and RingOffset are ring-2 ring-offset-1 on buttons, toggles,
	// tabs, checkboxes and switches: a 1 DIP gap in the background color, then
	// a 2 DIP ring.
	RingWidth  float32 = 2
	RingOffset float32 = 1
	// InputRingWidth is a restrained 2 DIP outer focus ring on text controls, at InputRingAlpha.
	InputRingWidth float32 = 2
	InputRingAlpha float32 = 0.24
	// InvalidBorderAlpha is the destructive/64 border of an invalid control;
	// InvalidRingAlphaLight/Dark its destructive/16 (/24 dark) focus ring.
	InvalidBorderAlpha    float32 = 0.64
	InvalidRingAlphaLight float32 = 0.16
	InvalidRingAlphaDark  float32 = 0.24
	// BorderWidth is the 1 DIP `border`.
	BorderWidth float32 = 1
)

// Opacity rules (tokens.json opacity).
const (
	OpacityDisabled    float32 = 0.64
	OpacityPlaceholder float32 = 0.72
	OpacityIcon        float32 = 0.80
)

// Icon sizes (size-*): the desktop default svg size and its neighbours.
const (
	IconXS float32 = 14
	IconSM float32 = 16
	IconMD float32 = 18
	IconLG float32 = 20
)

// CubicBezier is a CSS cubic-bezier timing function.
type CubicBezier struct{ X1, Y1, X2, Y2 float32 }

// At returns the eased progress at time fraction t, solving the curve's x
// for t as browsers do.
func (b CubicBezier) At(t float32) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	if b.X1 == b.Y1 && b.X2 == b.Y2 {
		return t
	}
	x1, y1, x2, y2 := float64(b.X1), float64(b.Y1), float64(b.X2), float64(b.Y2)
	cx := 3 * x1
	bx := 3*(x2-x1) - cx
	ax := 1 - cx - bx
	cy := 3 * y1
	by := 3*(y2-y1) - cy
	ay := 1 - cy - by
	x := float64(t)
	u := x
	for i := 0; i < 8; i++ {
		fx := ((ax*u+bx)*u+cx)*u - x
		d := (3*ax*u+2*bx)*u + cx
		if math.Abs(fx) < 1e-6 {
			break
		}
		if math.Abs(d) < 1e-6 {
			break
		}
		u -= fx / d
	}
	if u < 0 || u > 1 || math.Abs(((ax*u+bx)*u+cx)*u-x) > 1e-4 {
		lo, hi := 0.0, 1.0
		u = x
		for i := 0; i < 30; i++ {
			v := ((ax*u+bx)*u + cx) * u
			if math.Abs(v-x) < 1e-6 {
				break
			}
			if v < x {
				lo = u
			} else {
				hi = u
			}
			u = (lo + hi) / 2
		}
	}
	return float32(((ay*u+by)*u + cy) * u)
}

// Easings as ui.Easing values, made once so views never allocate them.
var (
	EaseInOut    ui.Easing = CurveEaseInOut.At
	EaseOut      ui.Easing = CurveEaseOut.At
	EaseIn       ui.Easing = CurveEaseIn.At
	EaseDrawer   ui.Easing = CurveDrawer.At
	EaseLinear   ui.Easing = CurveLinear.At
	EaseToastOut ui.Easing = CurveToastSlide.At
)

// Transition is a part's CSS transition: the properties it animates, how
// long and along which curve.
type Transition struct {
	Props    string
	Duration time.Duration
	Curve    CubicBezier
}

// Durations coss uses that are not a part's transition.
const (
	DurationPanel    = 200 * time.Millisecond // dialogs, sheets, tabs, accordion height
	DurationDrawer   = 450 * time.Millisecond
	DurationProgress = 500 * time.Millisecond
	DurationSpin     = time.Second             // animate-spin
	DurationSkeleton = 2600 * time.Millisecond // animate-skeleton
	DurationPulse    = 2 * time.Second         // animate-pulse
	DurationCaret    = time.Second             // caret-blink
)

// TransitionFor returns the transition of a coss part ("button/button",
// "tabs/tab-indicator"), or the 150 ms ease-in-out default when the part
// declares none.
func TransitionFor(part string) Transition {
	if t, ok := transitions[part]; ok {
		return t
	}
	return Transition{Props: "none", Duration: DurationDefault, Curve: CurveEaseInOut}
}

// Delta is what a part animates from as it opens (Base UI's
// data-starting-style) or to as it closes (data-ending-style).
type Delta struct {
	// Opacity, when HasOpacity, is the opacity there.
	Opacity    float32
	HasOpacity bool
	// Scale is the scale there; 0 leaves the size.
	Scale float32
	// TranslateX/Y move by DIPs; SlideX/Y by the part's own width/height
	// (±1 is a full slide off its edge).
	TranslateX, TranslateY float32
	SlideX, SlideY         float32
	// Collapse animates the height to 0 (accordion, collapsible panels).
	Collapse bool
	// Raw lists the declarations a native view cannot take as numbers.
	Raw string
}

// EnterOf and ExitOf return the part's enter and exit deltas.
func EnterOf(part string) Delta { return deltas[part+"|enter"] }

// ExitOf returns what the part animates to as it closes.
func ExitOf(part string) Delta { return deltas[part+"|exit"] }

// Motion converts the delta to a ui.Motion for ElementTransition, given the
// part's size for slides.
func (d Delta) Motion(w, h float32) ui.Motion {
	m := ui.Motion{X: d.TranslateX + d.SlideX*w, Y: d.TranslateY + d.SlideY*h, Collapse: d.Collapse, Scale: d.Scale, Opacity: 1}
	if d.HasOpacity {
		m.Opacity = d.Opacity
	}
	return m
}
