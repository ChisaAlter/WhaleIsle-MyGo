package interaction

import (
	"time"
	"whaleisle/internal/dsh"
)

// MatchesClient prevents a child pane from attaching requests owned by a
// different Host during desktop endpoint replacement.
func (s *Store) MatchesClient(client *dsh.Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return client != nil && s.client == client
}

// SyncQuestionFocusForAgent releases only this conversation's pristine pause.
// A separately rendered parent/child pane cannot clear another pane's focus.
func (s *Store) SyncQuestionFocusForAgent(agentID, eventID string) {
	if agentID == "" {
		return
	}
	s.mu.Lock()
	if s.questionFocusFrame {
		if eventID != "" || s.questionFocusScopes[agentID] == "" {
			s.questionFocusScopes[agentID] = eventID
		}
		s.mu.Unlock()
		return
	}
	changed := false
	now := time.Now()
	for _, p := range s.pending {
		if p.AgentID != agentID || p.Event != EventUserQuestions {
			continue
		}
		ds := s.questionDrafts[p.EventID]
		if ds == nil {
			continue
		}
		on := p.EventID == eventID && !p.Sending && !p.Continued && !ds.hidden
		changed = ds.wait.focus(on, now) || changed
	}
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}

// Frame aggregation handles duplicate child panes and releases hidden scopes.
func (s *Store) BeginQuestionFocusFrame() {
	s.mu.Lock()
	s.questionFocusFrame = true
	s.questionFocusScopes = map[string]string{}
	s.mu.Unlock()
}
func (s *Store) EndQuestionFocusFrame() {
	s.mu.Lock()
	s.questionFocusFrame = false
	changed := false
	now := time.Now()
	for _, p := range s.pending {
		ds := s.questionDrafts[p.EventID]
		if ds == nil || p.Event != EventUserQuestions {
			continue
		}
		on := s.questionFocusScopes[p.AgentID] == p.EventID && !p.Sending && !p.Continued && !ds.hidden
		changed = ds.wait.focus(on, now) || changed
	}
	s.questionFocusScopes = nil
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}
