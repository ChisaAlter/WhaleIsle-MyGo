package interaction

import (
	"context"
	"time"

	"whaleisle/internal/dsh"
)

// QuestionWaitView describes the foreground claim, not a guessed Host deadline.
type QuestionWaitView struct {
	RemainingMs int64
	Running     bool
	State       string
}

type questionWaitState struct {
	active, focused, editing, held bool
	deadline                       time.Time
	remaining                      time.Duration
	generation                     uint64
	wake                           chan struct{}
}

func (w *questionWaitState) signal() {
	if w.wake != nil {
		select {
		case w.wake <- struct{}{}:
		default:
		}
	}
}
func (w *questionWaitState) view(now time.Time) *QuestionWaitView {
	if !w.active {
		return nil
	}
	remaining := w.remaining
	if !w.deadline.IsZero() {
		remaining = max(0, w.deadline.Sub(now))
	}
	state := "counting"
	if w.held {
		state = "waiting"
	} else if w.editing {
		state = "editing"
	} else if w.focused {
		state = "focused"
	}
	return &QuestionWaitView{RemainingMs: remaining.Milliseconds(), Running: !w.deadline.IsZero(), State: state}
}
func (w *questionWaitState) pause(now time.Time) {
	if !w.deadline.IsZero() {
		w.remaining = max(0, w.deadline.Sub(now))
		w.deadline = time.Time{}
	}
}
func (w *questionWaitState) focus(on bool, now time.Time) bool {
	if w.focused == on || w.editing || w.held {
		return false
	}
	w.focused = on
	if on {
		w.pause(now)
	} else if w.active {
		w.deadline = now.Add(w.remaining)
	}
	w.signal()
	return true
}
func (s *Store) engageQuestionLocked(eventID string) {
	ds := s.questionDrafts[eventID]
	p := s.findPending(eventID)
	if ds == nil || p == nil || p.Sending || p.Continued {
		return
	}
	ds.wait.pause(time.Now())
	ds.wait.editing, ds.wait.focused = true, false
	ds.wait.signal()
}

// SyncQuestionFocus has one desktop keyboard-focus owner. Switching sessions,
// hiding the answer body or blurring the window releases a pristine pause.
func (s *Store) SyncQuestionFocus(agentID, eventID string) {
	s.mu.Lock()
	changed := false
	now := time.Now()
	for _, p := range s.pending {
		ds := s.questionDrafts[p.EventID]
		if ds == nil || p.Event != EventUserQuestions {
			continue
		}
		on := p.AgentID == agentID && p.EventID == eventID && !p.Sending && !p.Continued && !ds.hidden
		changed = ds.wait.focus(on, now) || changed
	}
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}

// TakeQuestionTime holds this exact live request until the user answers.
func (s *Store) TakeQuestionTime(agentID, eventID string) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	p, ds := s.findPending(eventID), s.questionDrafts[eventID]
	if p == nil || p.AgentID != agentID || p.Event != EventUserQuestions || p.Sending || p.Continued || ds == nil || !ds.wait.active || ds.wait.held {
		s.mu.Unlock()
		return
	}
	ds.wait.pause(time.Now())
	ds.wait.held, ds.wait.focused = true, false
	ds.wait.signal()
	s.mu.Unlock()
	s.fire()
}

// runQuestionWait owns countdown repaint and timeout settlement. The request,
// draft, generation and waiter are revalidated while holding the same lock as
// focus/edit changes, so a queued timer cannot race a hold into a timeout.
func (s *Store) runQuestionWait(ctx context.Context, agentID, callID, eventID string, remaining time.Duration) {
	s.mu.Lock()
	p, ds := s.findPending(eventID), s.questionDrafts[eventID]
	if ctx.Err() != nil || p == nil || p.AgentID != agentID || p.CallID != callID || p.Event != EventUserQuestions || p.Continued || ds == nil {
		s.mu.Unlock()
		return
	}
	w := &ds.wait
	w.active = true
	w.remaining = remaining
	w.generation++
	generation := w.generation
	w.wake = make(chan struct{}, 1)
	wake := w.wake
	if !w.held && !w.editing && !w.focused {
		w.deadline = time.Now().Add(remaining)
	}
	s.mu.Unlock()
	s.fire()
	defer func() {
		s.mu.Lock()
		current := s.questionDrafts[eventID] == ds && w.generation == generation
		if current {
			w.active = false
			w.deadline = time.Time{}
		}
		s.mu.Unlock()
		if current {
			s.fire()
		}
	}()
	for {
		s.mu.Lock()
		p = s.findPending(eventID)
		if ctx.Err() != nil || p == nil || p.AgentID != agentID || p.CallID != callID || p.Continued || s.questionDrafts[eventID] != ds || w.generation != generation {
			s.mu.Unlock()
			return
		}
		delay := time.Duration(0)
		if !p.Sending && !w.deadline.IsZero() {
			delay = time.Until(w.deadline)
			if delay <= 0 {
				ch := s.waiters[eventID]
				if ch != nil {
					outcome := dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "UserQuestionError", Code: "ASK_TIMED_OUT", Message: "Question timed out"}}
					p.Sending, p.Error = true, ""
					s.responses[eventID] = outcome
					ch <- outcome
				}
				s.mu.Unlock()
				s.fire()
				return
			}
			delay = min(delay, time.Second)
		}
		s.mu.Unlock()
		if delay == 0 {
			select {
			case <-ctx.Done():
				return
			case <-wake:
			}
			continue
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-wake:
			timer.Stop()
		case <-timer.C:
			s.fire()
		}
	}
}
