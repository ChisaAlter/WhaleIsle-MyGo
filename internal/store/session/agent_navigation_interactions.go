package session

import (
	"encoding/json"
	"whaleisle/internal/dsh"
)

// AgentInteractionTarget authorizes only the current live continuable child.
// One-shot/external, historical pages, restored panes and stale tokens remain
// read-only even if another conversation has a pending waterfall request.
func (s *Store) AgentInteractionTarget(token uint64, address AgentSessionAddress) (*dsh.Client, QuestionProjectionState, []json.RawMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &s.agentNavigation
	if !s.agentPaneCurrentLocked() || !s.agentNavigationOwnedLocked() || n.view.Token != token || n.view.Address != address || !address.validChild() || address.Mode != "continuable" || n.view.Busy || !n.view.Process.Live {
		return nil, QuestionProjectionState{}, nil, false
	}
	return s.agentInteractionProjectionLocked(address)
}

// Caller has already checked this exact pane and holds its lock.
func (s *Store) agentInteractionProjectionLocked(address AgentSessionAddress) (*dsh.Client, QuestionProjectionState, []json.RawMessage, bool) {
	n := &s.agentNavigation
	p := s.questionProjections[address.ChildSessionID]
	if owner := s.agentNavigationOwner; owner != nil {
		owner.mu.Lock()
		other := owner.questionProjections[address.ChildSessionID]
		if other.Questions != nil && (p.Questions == nil || other.QuestionsSeq >= p.QuestionsSeq) {
			p.Questions, p.QuestionsSeq = other.Questions, other.QuestionsSeq
		}
		if other.Inbox != nil && (p.Inbox == nil || other.InboxSeq >= p.InboxSeq) {
			p.Inbox, p.InboxSeq = other.Inbox, other.InboxSeq
		}
		p.Questions = append(json.RawMessage(nil), p.Questions...)
		p.Inbox = append(json.RawMessage(nil), p.Inbox...)
		owner.mu.Unlock()
	} else {
		p.Questions = append(json.RawMessage(nil), p.Questions...)
		p.Inbox = append(json.RawMessage(nil), p.Inbox...)
	}
	var records []json.RawMessage
	for _, m := range n.view.Messages {
		if len(m.Raw) > 0 && !m.Streaming {
			records = append(records, append(json.RawMessage(nil), m.Raw...))
		}
	}
	return n.client, p, records, true
}
