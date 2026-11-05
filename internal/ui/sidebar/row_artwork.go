package sidebar

import (
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

// groupChevronAngle eases a Workspace row's disclosure triangle towards the
// group's open state. The caller redraws while the angle is still moving, so
// collapsing a group spins its arrow instead of swapping glyphs.
func groupChevronAngle(c *ui.Context, owner ui.Element, collapsed bool) float32 {
	state := ui.Local(owner, "workspace-chevron", func() chevronAnimation { return chevronAnimation{} })
	target := float32(0)
	if !collapsed {
		target = 90
	}
	if !state.started {
		state.started, state.angle, state.to = true, target, target
	}
	if state.to != target {
		state.from, state.to, state.since = state.angle, target, c.Now()
	}
	if state.angle != state.to {
		progress := float32(c.Now().Sub(state.since)) / float32(chevronSettle)
		if progress >= 1 {
			state.angle = state.to
		} else {
			// Ease-out cubic, the settle the original's arrow uses.
			inverted := 1 - progress
			state.angle = state.from + (state.to-state.from)*(1-inverted*inverted*inverted)
			c.After(16 * time.Millisecond)
		}
	}
	return state.angle
}
