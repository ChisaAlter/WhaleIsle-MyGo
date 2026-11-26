package design

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Shared 5-tier motion duration tokens matching ui-theme/src/styles/base.css:13-20
// and docs/motion.md. All standard UI motion rides EaseInOut, which resolves
// CurveEaseInOut = cubic-bezier(0.4, 0, 0.2, 1).
const (
	MotionFast    = 100 * time.Millisecond // swap, switch, press, overlay-out
	MotionPopover = 160 * time.Millisecond // menu/popover, card hover
	MotionDefault = 200 * time.Millisecond // overlay-in, collapse, dock slide, label track
	MotionSlow    = 300 * time.Millisecond // shell columns, draft->session track
	MotionFlip    = 400 * time.Millisecond // FlipText rotateX, queue system row settle
)

// ButtonMarqueeMinPeriod is the minimum period of a button busy/feedback overflow
// marquee animation (ui-primitives/src/Button.tsx:94).
const ButtonMarqueeMinPeriod = 3600 * time.Millisecond

// ButtonPressOffset animates subtle press displacement (0..1 DIP) over MotionFast (100ms)
// along EaseInOut, returning 0 immediately when disabled or ReduceMotion is set.
func ButtonPressOffset(c *ui.Context, host ui.Element, pressed, disabled bool) float32 {
	if c == nil || host == (ui.Element{}) {
		return 0
	}
	target := float32(0)
	if pressed && !disabled && !c.Preferences().ReduceMotion {
		target = 1
	}
	return host.AnimateWith("design.press", target, MotionFast, EaseInOut)
}

// ButtonMarqueeDuration returns the ping-pong scroll period for overflow distance d (in DIPs),
// clamped to at least 3.6s (Button.tsx:94: max(3600, dist/24*2000 + 1600) ms).
func ButtonMarqueeDuration(distance float32) time.Duration {
	if distance <= 0 {
		return ButtonMarqueeMinPeriod
	}
	ms := int64(distance/24*2000 + 1600)
	d := time.Duration(ms) * time.Millisecond
	if d < ButtonMarqueeMinPeriod {
		return ButtonMarqueeMinPeriod
	}
	return d
}

// ButtonMarqueeOffset computes the horizontal translateX offset for elapsed animation time
// inside a clipped button feedback viewport (Button.module.css:145-148):
// 0%,15%,90%,100% -> 0; 45%,60% -> -distance. Returns 0 under ReduceMotion or <=1 DIP overflow.
func ButtonMarqueeOffset(elapsed time.Duration, distance float32, reduceMotion bool) float32 {
	if reduceMotion || distance <= 1 || elapsed <= 0 {
		return 0
	}
	period := ButtonMarqueeDuration(distance)
	phase := float32(elapsed%period) / float32(period)
	switch {
	case phase <= 0.15, phase >= 0.90:
		return 0
	case phase < 0.45:
		return -distance * ((phase - 0.15) / 0.30)
	case phase <= 0.60:
		return -distance
	default:
		return -distance * (1 - (phase-0.60)/0.30)
	}
}

type marqueeState struct {
	text      string
	started   time.Time
	viewportW float32
}

// ButtonMarqueeLabel renders a transient busy/feedback label inside a clipped
// overlay viewport without growing its button. When busy is true an ongoing dot
// leads the copy; when label exceeds the viewport width it ping-pongs
// horizontally over >=3.6s (staying static under ReduceMotion).
func ButtonMarqueeLabel(c *ui.Context, host ui.Element, label string, fontSize, lineHeight float32, weight int, fg ui.Color, busy bool) {
	row := ui.Row(c).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Center).Justify(ui.Center).PassThrough()
	row.Children(func() {
		if busy {
			ui.Box(c).Size(8, 8).Radius(RadiusFull).Background(fg).Shrink(0)
		}
		if label == "" {
			return
		}
		st := ui.Local(host, "design.marquee", func() marqueeState { return marqueeState{} })
		now := c.Now()
		if label != st.text {
			st.text = label
			st.started = now
		}
		var textW float32
		if fontSize > 0 {
			textW, _ = ui.ShapeTextLine(label, ui.Font{Family: c.Theme().Font, Size: fontSize, Weight: weight}, 1e6).Size()
		}
		overflows := st.viewportW > 0 && textW-st.viewportW > 1
		vp := ui.Row(c).Grow(1).MinWidth(0).ClipX().AlignItems(ui.Center).PassThrough()
		if !overflows {
			vp.Justify(ui.Center)
		}
		vp.ObserveLayout(func(r ui.Rect) {
			if r.W > 0 && st.viewportW != r.W {
				st.viewportW = r.W
			}
		})
		vp.Children(func() {
			tx := ui.Text(c, label).FontSize(fontSize).FixedLineHeight(lineHeight).FontWeight(weight).TextColor(fg).SingleLine().Shrink(0)
			if st.viewportW == 0 && !now.IsZero() && !c.Preferences().ReduceMotion {
				c.AnimationFrame()
			}
			if overflows && !now.IsZero() && !st.started.IsZero() && !c.Preferences().ReduceMotion {
				dist := textW - st.viewportW
				off := ButtonMarqueeOffset(now.Sub(st.started), dist, false)
				tx.OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{X: off} })
				c.AnimationFrame()
			}
		})
	})
}
