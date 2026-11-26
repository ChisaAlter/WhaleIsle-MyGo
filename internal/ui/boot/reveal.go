package boot

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/widgets"
)

// RevealDuration matches boot-reveal.css: 3 * --ds-transition-duration (600ms).
const RevealDuration = 3 * widgets.OverlayDuration

// RevealCurtains splits the boot backdrop along the 62% horizon and slides
// the sky upward (-100%) and sea downward (+100%) over 600ms after cold
// startup hands off to the workbench, stopping at 0 Hz once complete.
type RevealCurtains struct {
	active  bool
	started time.Time
	dark    bool
	failed  bool
}

// Trigger arms the 600ms horizon split reveal unless reduced motion or minimal
// mode disables decorative transitions.
func (r *RevealCurtains) Trigger(dark, failed, reduceMotion bool) {
	if reduceMotion || renderpolicy.Minimal() {
		r.active = false
		return
	}
	r.active = true
	r.started = time.Time{}
	r.dark = dark
	r.failed = failed
}

// Active reports whether the reveal split is still running.
func (r *RevealCurtains) Active() bool { return r.active }

// Reset clears any pending reveal.
func (r *RevealCurtains) Reset() {
	r.active = false
	r.started = time.Time{}
}

// Paint renders the split curtains over the workbench while active and retires
// immediately (0 Hz idle) when elapsed >= RevealDuration or under reduced
// motion / minimal mode.
func (r *RevealCurtains) Paint(c *ui.Context) {
	if !r.active {
		return
	}
	if c.Preferences().ReduceMotion || renderpolicy.Minimal() {
		r.Reset()
		return
	}
	if now := c.Now(); !now.IsZero() && !r.started.IsZero() && now.Sub(r.started) >= RevealDuration {
		r.Reset()
		return
	}
	radius := bootRadius(c)
	ui.Box(c).Key("boot-reveal-curtains").Absolute().Fill().PassThrough().Draw(func(p *ui.Painter, rect ui.Rect) {
		if !r.active {
			return
		}
		now := p.Now()
		if now.IsZero() {
			r.Reset()
			return
		}
		if r.started.IsZero() || r.started.After(now) {
			r.started = now
		}
		elapsed := now.Sub(r.started)
		if elapsed >= RevealDuration {
			r.Reset()
			return
		}
		progress := widgets.DesignEase(float32(elapsed) / float32(RevealDuration))
		horizonY := rect.Y + rect.H*sceneHorizon
		skyH := rect.H * sceneHorizon
		seaH := rect.H * (1 - sceneHorizon)
		skyShift := -skyH * progress
		seaShift := seaH * progress

		p.Clip(rect, radius, func() {
			if skyClipH := skyH + skyShift; skyClipH > 0 {
				skyClip := ui.Rect{X: rect.X, Y: rect.Y, W: rect.W, H: skyClipH}
				p.Clip(skyClip, 0, func() {
					bootScene(r.dark, r.failed, false)(p, ui.Rect{X: rect.X, Y: rect.Y + skyShift, W: rect.W, H: rect.H})
				})
			}
			seaTop := horizonY + seaShift
			if seaClipH := rect.Y + rect.H - seaTop; seaClipH > 0 {
				seaClip := ui.Rect{X: rect.X, Y: seaTop, W: rect.W, H: seaClipH}
				p.Clip(seaClip, 0, func() {
					bootScene(r.dark, r.failed, false)(p, ui.Rect{X: rect.X, Y: rect.Y + seaShift, W: rect.W, H: rect.H})
				})
			}
		})
		p.AnimationFrame()
	})
}
