package boot

import (
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/theme"
)

type curtainState struct {
	Seen   bool
	Active bool
	Cycle  uint64
}

func curtainRects(r ui.Rect, progress float32) (ui.Rect, ui.Rect) {
	progress = max(0, min(1, progress))
	skyHeight := r.H * .62
	seaHeight := r.H - skyHeight
	return ui.Rect{X: r.X, Y: r.Y - skyHeight*progress, W: r.W, H: skyHeight}, ui.Rect{X: r.X, Y: r.Y + skyHeight + seaHeight*progress, W: r.W, H: seaHeight}
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
	progress := anchor.Animate(state.Cycle, target, 600*time.Millisecond)
	if !state.Seen {
		return
	}
	if !active && m.Status.State != harness.Ready {
		state.Seen = false
		return
	}
	if !active && progress >= 1 {
		state.Seen = false
		return
	}
	if active {
		progress = 0
	}
	th := theme.Resolve(m.ThemeID)
	dark := th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark)
	ui.Overlay(c, func() {
		w, h := c.Size()
		top := float32(48)
		surface := ui.Column(c.Key("boot-curtain-surface")).Absolute().Left(0).Top(top).Width(w).Height(max(0, h-top))
		if active {
			surface.Modal()
		} else {
			surface.PassThrough()
		}
		surface.Draw(func(p *ui.Painter, r ui.Rect) { paintBootBackdrop(p, r, progress, dark) })
		if active {
			surface.Padding(24).Children(func() {
				ui.Box(c).Height(max(0, (h-top)*.15))
				startupScene(c, m)
			})
		}
	})
}

func paintBootBackdrop(p *ui.Painter, r ui.Rect, progress float32, dark bool) {
	sky, sea := curtainRects(r, progress)
	top, horizon, bottom := ui.RGB(241, 246, 248), ui.RGB(204, 227, 235), ui.RGB(84, 128, 152)
	if dark {
		top, horizon, bottom = ui.RGB(2, 4, 9), ui.RGB(12, 34, 48), ui.RGB(4, 13, 23)
	}
	p.Clip(r, 0, func() {
		p.FillGradient(sky, ui.LinearGradient{From: top, To: horizon, Angle: 180}, 0)
		p.FillGradient(sea, ui.LinearGradient{From: horizon, To: bottom, Angle: 180}, 0)
		p.Clip(sky, 0, func() {
			// Stable stars have no timers and cannot prolong Ready's finite reveal.
			if dark {
				for i := 0; i < 76; i++ {
					x := sky.X + float32((i*137+29)%997)/997*sky.W
					y := sky.Y + float32((i*89+17)%613)/613*sky.H
					size := float32(1)
					if i%13 == 0 {
						size = 2.5
					}
					p.Fill(ui.Rect{X: x, Y: y, W: size, H: size}, ui.RGBA(190, 220, 240, .45), size/2)
				}
			}
		})
		p.Clip(sea, 0, func() {
			p.Line(sea.X, sea.Y, sea.X+sea.W, sea.Y, 1, ui.RGBA(190, 224, 240, .4))
			for i := 0; i < 30; i++ {
				x := sea.X + float32((i*173+41)%991)/991*sea.W
				y := sea.Y + float32((i*83+11)%397)/397*sea.H
				p.Fill(ui.Rect{X: x, Y: y, W: 2, H: 2}, ui.RGBA(175, 212, 230, .3), 1)
			}
		})
	})
}
