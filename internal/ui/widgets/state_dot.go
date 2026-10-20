package widgets

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// State is what a tracked thing is doing, as a StateDot shows it in one
// glance: nothing in progress, work running, finished, asking for attention,
// or failed.
type State uint8

const (
	// StateIdle is a subject with nothing in progress.
	StateIdle State = iota
	// StateRunning is work in progress: the dot breathes.
	StateRunning
	// StateDone is work that went well.
	StateDone
	// StateWarning is work that needs attention, as a capped or partial run.
	StateWarning
	// StateError is work that failed.
	StateError
)

// String names the state, for tests and debugging.
func (s State) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateDone:
		return "done"
	case StateWarning:
		return "warning"
	case StateError:
		return "error"
	}
	return "idle"
}

// StateDotSlot is the width and height a StateDot reserves, whatever state it
// shows, so that rows of dots line up.
const StateDotSlot = 10

// stateCore is the diameter of the solid dot inside the slot.
const stateCore = 6

// StateColor returns the color the theme gives a state, for the text beside a
// dot as well as for the dot itself.
func StateColor(t *ui.Theme, s State) ui.Color {
	switch s {
	case StateRunning:
		return t.TextMuted
	case StateDone:
		return t.Success
	case StateWarning:
		return t.Warning
	case StateError:
		return t.Danger
	}
	return t.TextMuted.Alpha(0.8)
}

// StateDot shows a state in a 10 DIP slot: five states, each a solid 6 DIP
// dot, except running, whose dot carries a ring that breathes while work goes
// on. It is decorative, so assistive technology skips it: name the state in
// the text beside it, as "Running" or "Failed".
//
//	widgets.StateDot(c, widgets.StateRunning)
//
// The breathing respects the desktop's reduce-motion setting: without motion
// the ring stays still.
func StateDot(c *ui.Context, s State) ui.Element {
	t := c.Theme()
	color := StateColor(t, s)
	dot := ui.Box(c).Size(StateDotSlot, StateDotSlot).Shrink(0).Role(ui.RoleNone)
	pulse := float32(0.35)
	if s == StateRunning && !c.Preferences().ReduceMotion {
		// Out and back every 1.5s, and a frame for every step of it.
		pulse = dot.Loop("state-dot", 1500*time.Millisecond, ui.Bounce(ui.EaseInOut))
	}
	ring := 4.5 + 0.45*pulse
	alpha := 0.55 - 0.4*pulse
	dot.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		if s == StateRunning {
			// The stroke is inset from the ring's edge, which keeps it
			// outside the solid dot.
			p.Stroke(ui.Rect{X: cx - ring, Y: cy - ring, W: ring * 2, H: ring * 2}, color.Alpha(alpha), ring, 1.2)
		}
		p.Fill(ui.Rect{X: cx - stateCore/2, Y: cy - stateCore/2, W: stateCore, H: stateCore}, color, stateCore/2)
	})
	return dot
}
