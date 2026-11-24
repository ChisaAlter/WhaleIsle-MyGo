package sidebar

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Row artwork transcribed from the original's regular-weight icon set: the
// Workspace rows show a folder at rest and a filled disclosure triangle while
// the pointer rests on them (Rows.module.css .projectRow .folder/.chevron).
var (
	workspaceFolderArtwork  = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1" stroke-linejoin="round"><path d="M2 4.3c0-.4.3-.75.75-.75h3.06c.2 0 .4.08.54.23l1.02 1.02h6.13c.41 0 .75.34.75.75v6.58c0 .41-.34.75-.75.75H2.75A.75.75 0 0 1 2 12.13V4.3Z"/></svg>`))
	workspaceChevronArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="currentColor"><path d="M6 4.1 11.2 8 6 11.9V4.1Z"/></svg>`))
	archiveArtwork          = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1" stroke-linejoin="round" stroke-linecap="round"><rect x="2.2" y="3" width="11.6" height="2.9" rx=".7"/><path d="M3.2 5.9v6.1c0 .5.4.9.9.9h7.8c.5 0 .9-.4.9-.9V5.9"/><path d="M6.2 8.4h3.6"/></svg>`))
	pinArtwork              = sessionPinArtwork
)

// chevronSettle is the original's 300 ms arrow rotation
// (--ds-transition-duration-slow) between "closed" (0°) and "open" (90°).
const chevronSettle = 300 * time.Millisecond

type chevronAnimation struct {
	angle, from, to float32
	since           time.Time
	started         bool
}

func (s *chevronAnimation) update(now time.Time, collapsed bool) {
	target := float32(0)
	if !collapsed {
		target = 90
	}
	if !s.started {
		s.started, s.angle, s.from, s.to = true, target, target, target
		return
	}
	if s.to != target {
		s.from = s.eval(now)
		s.to = target
		s.since = now
	}
}

func (s *chevronAnimation) eval(now time.Time) float32 {
	if s.since.IsZero() || s.from == s.to {
		s.angle = s.to
		return s.to
	}
	progress := float32(now.Sub(s.since)) / float32(chevronSettle)
	if progress <= 0 {
		s.angle = s.from
		return s.from
	}
	if progress >= 1 {
		s.angle = s.to
		s.from = s.to
		s.since = time.Time{}
		return s.to
	}
	inverted := 1 - progress
	s.angle = s.from + (s.to-s.from)*(1-inverted*inverted*inverted)
	return s.angle
}

func (s *chevronAnimation) moving(now time.Time) bool {
	return !s.since.IsZero() && s.from != s.to && now.Sub(s.since) < chevronSettle
}

// groupChevronAngle evaluates the current disclosure angle without scheduling a
// full-tree rebuild timer. Animated rotation is painted via groupChevronIcon.
func groupChevronAngle(c *ui.Context, owner ui.Element, collapsed bool) float32 {
	state := ui.Local(owner, "workspace-chevron", func() chevronAnimation { return chevronAnimation{} })
	now := c.Now()
	state.update(now, collapsed)
	return state.eval(now)
}

// groupChevronIcon paints the disclosure triangle with paint-only p.After
// animation so expanding/collapsing a workspace never triggers 60fps tree rebuilds.
func groupChevronIcon(c *ui.Context, owner ui.Element, collapsed bool, size float32, ink ui.Color, label string) ui.Element {
	state := ui.Local(owner, "workspace-chevron", func() chevronAnimation { return chevronAnimation{} })
	reduceMotion := c.Preferences().ReduceMotion
	if reduceMotion {
		target := float32(0)
		if !collapsed {
			target = 90
		}
		state.started, state.angle, state.from, state.to, state.since = true, target, target, target, time.Time{}
	} else {
		state.update(c.Now(), collapsed)
	}
	return ui.Box(c).Size(size, size).Label(label).Draw(func(p *ui.Painter, r ui.Rect) {
		now := p.Now()
		deg := state.eval(now)
		rad := float64(deg) * math.Pi / 180.0
		cos, sin := float32(math.Cos(rad)), float32(math.Sin(rad))
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		sx, sy := r.W/16, r.H/16
		pts := [3][2]float32{
			{(6 - 8) * sx, (4.1 - 8) * sy},
			{(11.2 - 8) * sx, (8 - 8) * sy},
			{(6 - 8) * sx, (11.9 - 8) * sy},
		}
		var path ui.Path
		for i, pt := range pts {
			x := cx + pt[0]*cos - pt[1]*sin
			y := cy + pt[0]*sin + pt[1]*cos
			if i == 0 {
				path.MoveTo(x, y)
			} else {
				path.LineTo(x, y)
			}
		}
		path.Close()
		p.FillPath(&path, ink)
		if !reduceMotion && state.moving(now) {
			p.After(16 * time.Millisecond)
		}
	})
}
