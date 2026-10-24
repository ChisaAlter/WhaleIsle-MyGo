package boot

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

const (
	// curtainCaption is the shell strip above the boot page. The reference boot
	// page paints its .scene over the whole window content box: boot.css .scene
	// is height 100% and the 48px .caption is a transparent drag strip inside
	// that page (boot.css:64-72), so the reveal curtain covers the whole content
	// box — window.js:494 setBounds(win.getContentBounds()) plus boot-reveal.css
	// #dshd-boot-curtains inset:0. MyGo draws the caption strip as shell chrome
	// above the boot page (cmd/whaleisle/main.go paints the boot body under the
	// title bar), so the boot page itself starts below it; the curtain must cover
	// exactly the page it is standing in for, or its frozen scene would sit 48
	// DIP off the live one.
	curtainCaption = 48
	// curtainRelease is the reference release duration,
	// calc(var(--ds-transition-duration, .2s) * 3) (boot-reveal.css:6).
	curtainRelease = 600 * time.Millisecond
	// curtainSky is .dshd-boot-curtain-sky height: 62% (the sea is the rest,
	// boot-reveal.css:7-8).
	curtainSky = .62
)

type curtainState struct {
	Seen   bool
	Active bool
	Cycle  uint64
}

// curtainProgressHook samples the release progress of every painted frame.
// The shell leaves it nil; the timing test drives a deterministic clock through
// it (the animation state itself lives in ui.Local, not in this package).
var curtainProgressHook func(progress float32)

// curtainCoverage is the area the curtain stands in for: the boot page's box,
// from below the shell caption strip to the bottom of the window.
func curtainCoverage(w, h float32) ui.Rect {
	top := min(curtainCaption, max(0, h))
	return ui.Rect{X: 0, Y: top, W: max(0, w), H: max(0, h-top)}
}

// curtainRects splits the covered box into the sky (top 62%) and sea (bottom
// 38%) windows at progress: the sky slides up by its own height, the sea down
// by its own (translateY(-100%) / translateY(100%), boot-reveal.css:12-13).
func curtainRects(r ui.Rect, progress float32) (ui.Rect, ui.Rect) {
	progress = max(0, min(1, progress))
	skyHeight := r.H * curtainSky
	seaHeight := r.H - skyHeight
	return ui.Rect{X: r.X, Y: r.Y - skyHeight*progress, W: r.W, H: skyHeight},
		ui.Rect{X: r.X, Y: r.Y + skyHeight + seaHeight*progress, W: r.W, H: seaHeight}
}

// curtainShifts is how far the frozen scene moves with each window: a slice of
// a snapshot moves rigidly, so the scene is painted shifted and clipped to the
// moving window instead of being stretched into it.
func curtainShifts(r ui.Rect, progress float32) (sky, sea float32) {
	progress = max(0, min(1, progress))
	skyHeight := r.H * curtainSky
	return -skyHeight * progress, (r.H - skyHeight) * progress
}

// BootOverlay is finite: only the supervisor's Ready edge starts a reveal.
// The decorative exiting curtains neither capture input nor delay Host use.
func bootOverlay(c *ui.Context, m Model) {
	anchor := ui.Box(c.Key("boot-curtain-state")).Size(0, 0)
	state := ui.Local(anchor, "curtain", func() curtainState { return curtainState{} })
	active := m.Status.State == harness.Starting || m.Status.State == harness.Failed || m.Status.RecoveryActive || m.Status.RecoveryCancelled
	if active && !state.Active {
		state.Cycle++
		state.Seen = true
	}
	state.Active = active
	target := float32(0)
	if !active {
		target = 1
	}
	// The design bezier, cubic-bezier(.4,0,.2,1), not the framework's
	// symmetric ease: the reference release runs on --ds-ease-in-out
	// (boot-reveal.css:6). Reduced motion jumps to the target in one frame
	// (ui.AnimateWith), matching `transition: none` (boot-reveal.css:14).
	progress := anchor.AnimateWith(state.Cycle, target, curtainRelease, widgets.DesignEase)
	if !state.Seen {
		return
	}
	if !active && m.Status.State != harness.Ready {
		state.Seen = false
		return
	}
	if curtainProgressHook != nil {
		curtainProgressHook(progress)
	}
	if !active && progress >= 1 {
		state.Seen = false
		return
	}
	if active {
		progress = 0
	}
	th := theme.Resolve(m.ThemeID)
	// Same palette arguments boot.View passes to the same painter, so the frozen
	// scene is the page's scene, not a look-alike (boot.View: colors.Active +
	// bootBackdrop(bg, accent, m.SystemDark)).
	colors := th.Active(m.SystemDark)
	ui.Overlay(c, func() {
		w, h := c.Size()
		box := curtainCoverage(w, h)
		surface := ui.Column(c.Key("boot-curtain-surface")).Absolute().Left(box.X).Top(box.Y).Width(box.W).Height(box.H)
		if active {
			// The boot page's own actions live on this layer while it stands in
			// for the page, so the stage blocks the shell below it.
			surface.Modal()
		} else {
			// The release is decorative: pointer-events:none in the reference,
			// and it can never hold a click the desktop should get.
			surface.PassThrough()
		}
		surface.Draw(func(p *ui.Painter, r ui.Rect) {
			paintCurtainScene(p, r, progress, bootBackdrop(chrome.Color(colors.BG), chrome.Color(colors.Accent), m.SystemDark))
		})
		if active {
			surface.Padding(24).Children(func() {
				ui.Box(c).Height(max(0, box.H*.15))
				startupScene(c, m)
			})
		}
	})
}

// paintCurtainScene paints the boot page's own backdrop, split in two and
// moved as the reference curtains move. Each window shows a rigid slice of the
// scene: the scene is painted shifted by the window's translation and clipped
// to the window, so the horizon stays glued to the slice instead of being
// stretched into it. The scene is the boot page's painter over the boot page's
// rect, so the covered area matches what the page paints (see
// docs/review-20261022/round140-boot-reveal-g05.md for why this cannot be a
// page capture: the shell has no offscreen page snapshot).
func paintCurtainScene(p *ui.Painter, r ui.Rect, progress float32, backdrop func(*ui.Painter, ui.Rect)) {
	sky, sea := curtainRects(r, progress)
	skyShift, seaShift := curtainShifts(r, progress)
	shifted := func(dy float32) ui.Rect {
		return ui.Rect{X: r.X, Y: r.Y + dy, W: r.W, H: r.H}
	}
	// The moving windows leave the page box as they travel; the page box is the
	// limit, so nothing is painted over the shell caption strip above it.
	p.Clip(r, 0, func() {
		p.Clip(sky, 0, func() { backdrop(p, shifted(skyShift)) })
		p.Clip(sea, 0, func() { backdrop(p, shifted(seaShift)) })
	})
}
