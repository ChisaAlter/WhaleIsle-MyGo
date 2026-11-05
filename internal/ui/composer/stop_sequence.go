package composer

import (
	"github.com/egoist/mygo/ui"
	"time"
)

// The original fixed input sequence defaults to two independent Esc presses
// within 500ms. This belongs to the focused composer, never a window shortcut.
const stopSequenceInterval = 500 * time.Millisecond

type stopTarget struct {
	sessionID        string
	turn             float64
	generation, host uint64
}

type stopSequence struct {
	target   stopTarget
	deadline time.Time
}

func (s *stopSequence) reset() { *s = stopSequence{} }

func (s *stopSequence) press(target stopTarget, now time.Time, interval time.Duration) bool {
	matched := s.target == target && !s.deadline.IsZero() && !now.After(s.deadline)
	s.reset()
	if !matched {
		s.target, s.deadline = target, now.Add(interval)
	}
	return matched
}

func composerStopTarget(m Model) (stopTarget, bool) {
	snap := m.Snapshot
	if snap.StopSequenceMS < 1 || !snap.ClientReady || snap.SelectedID == "" || snap.StopTurnSeq <= 0 || m.OnStop == nil || m.StopBlocked || m.Editing && !m.Child || m.stopSuggestionsOpen || m.SkillsOpen || m.ChipEdit != nil && m.ChipEdit.Open || m.State != nil && m.State.ModelCommand.Visible(snap.SelectedID) {
		return stopTarget{}, false
	}
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID && item.Running {
			return stopTarget{sessionID: snap.SelectedID, turn: snap.StopTurnSeq, generation: snap.StopGeneration, host: snap.ComposerHostGeneration}, true
		}
	}
	return stopTarget{}, false
}

// Reconcile on every painted frame, including focus/window loss and lifecycle
// changes between key events. Each DraftClient owns exactly one input region.
func (c *DraftClient) syncStop(m Model, focused bool) {
	target, eligible := composerStopTarget(m)
	if !focused || !eligible || c.marked != nil || c.stop.target != target {
		c.stop.reset()
	}
}

// BindStopRegion mirrors the original chat/composer focus regions for ordinary
// controls, including the fallback TextArea. Native DraftClient keystrokes retain
// their local handler so candidate dismissal, edit cancellation and IME win.
func BindStopRegion(area ui.Element, client *DraftClient, m Model, transcript bool) {
	if client == nil {
		return
	}
	index := 0
	if transcript {
		index = 1
	}
	sequence := &client.regionStop[index]
	target, valid := composerStopTarget(m)
	if !area.FocusWithin() || !valid || sequence.target != target {
		sequence.reset()
	}
	area.ObserveUnhandledEscape(func(eligible bool) bool {
		if !eligible || !valid {
			sequence.reset()
			return false
		}
		if sequence.press(target, time.Now(), time.Duration(m.Snapshot.StopSequenceMS)*time.Millisecond) {
			m.OnStop()
		}
		return true
	})
}
