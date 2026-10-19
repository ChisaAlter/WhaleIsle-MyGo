package session

import (
	"bytes"
	"encoding/json"
	"sync/atomic"
	"whaleisle/internal/dsh"
)

// QuestionProjectionState preserves both authoritative faces needed for late replies.
// nil means not loaded; it must never be confused with an authoritative empty inbox.
type QuestionProjectionState struct {
	Questions    json.RawMessage
	Inbox        json.RawMessage
	QuestionsSeq float64
	InboxSeq     float64
	// Queue observations survive coalesced control frames, but are retained
	// only while that call is active or currently queued on this Host.
	QueueSeen map[string]uint64
}

// This clock contains no Host data. Unique process-local ordinals let a child
// pane merge observations from its own follow and its root's control reader.
var questionQueueClock atomic.Uint64

func cloneQuestionQueueSeen(in map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(in))
	for id, ordinal := range in {
		out[id] = ordinal
	}
	return out
}

func mergeQuestionQueueSeen(a, b map[string]uint64) map[string]uint64 {
	out := cloneQuestionQueueSeen(a)
	for id, ordinal := range b {
		out[id] = max(out[id], ordinal)
	}
	return out
}

func questionQueuedCalls(raw json.RawMessage) map[string]bool {
	var lanes map[string]json.RawMessage
	queued := map[string]bool{}
	if json.Unmarshal(raw, &lanes) != nil {
		return queued
	}
	for _, lane := range []string{"next-step", "next-turn"} {
		var rows []struct{ Source struct{ Kind, CallID string } }
		if json.Unmarshal(lanes[lane], &rows) != nil {
			continue
		}
		for _, row := range rows {
			if row.Source.Kind == "user-question-reply" && row.Source.CallID != "" {
				queued[row.Source.CallID] = true
			}
		}
	}
	return queued
}

func (p *QuestionProjectionState) pruneQueueSeen() {
	var questions struct {
		Active []struct {
			CallID string `json:"callId"`
		} `json:"active"`
	}
	if json.Unmarshal(p.Questions, &questions) != nil || questions.Active == nil {
		return
	}
	retained := questionQueuedCalls(p.Inbox)
	for _, call := range questions.Active {
		retained[call.CallID] = true
	}
	for id := range p.QueueSeen {
		if !retained[id] {
			delete(p.QueueSeen, id)
		}
	}
}

// QuestionQueueObservation reads producer state at the actual submit gesture,
// before an in-flight attempt is installed. It is called outside interaction's
// lock and returns a clone bound to this actual client, never global history.
func (s *Store) QuestionQueueObservation(agentID string) (*dsh.Client, json.RawMessage, float64, map[string]uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.agentPaneCurrentLocked() {
		return nil, nil, 0, nil
	}
	p := s.questionProjections[agentID]
	seen := cloneQuestionQueueSeen(p.QueueSeen)
	if owner := s.agentNavigationOwner; owner != nil {
		owner.mu.Lock()
		if owner.client == s.client {
			other := owner.questionProjections[agentID]
			seen = mergeQuestionQueueSeen(seen, other.QueueSeen)
			if other.Inbox != nil && (p.Inbox == nil || other.InboxSeq >= p.InboxSeq) {
				p.Inbox, p.InboxSeq = other.Inbox, other.InboxSeq
			}
		}
		owner.mu.Unlock()
	}
	return s.client, append(json.RawMessage(nil), p.Inbox...), p.InboxSeq, seen
}

func (s *Store) questionProjectionLocked() QuestionProjectionState {
	p := s.questionProjections[s.selectedID]
	p.Questions = append(json.RawMessage(nil), p.Questions...)
	p.Inbox = append(json.RawMessage(nil), p.Inbox...)
	p.QueueSeen = cloneQuestionQueueSeen(p.QueueSeen)
	return p
}

func (s *Store) applyQuestionValuesLocked(values json.RawMessage, seq float64) {
	s.applyQuestionValuesForSessionLocked(s.selectedID, values, seq)
}
func (s *Store) applyQuestionValuesForSessionLocked(id string, values json.RawMessage, seq float64) {
	var faces map[string]json.RawMessage
	if id == "" || json.Unmarshal(values, &faces) != nil {
		return
	}
	for _, key := range []string{"userQuestions", "inbox"} {
		if raw, ok := faces[key]; ok {
			s.applyQuestionFaceLocked(id, key, raw, seq)
		}
	}
}
func (s *Store) applyQuestionFaceLocked(id, key string, raw json.RawMessage, seq float64) {
	if id == "" || (key != "userQuestions" && key != "inbox") || !json.Valid(raw) {
		return
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if key == "userQuestions" {
			raw = json.RawMessage(`{"active":[],"settled":[]}`)
		} else {
			raw = json.RawMessage(`{"next-step":[],"next-turn":[]}`)
		}
	}
	if s.questionProjections == nil {
		s.questionProjections = make(map[string]QuestionProjectionState)
	}
	p := s.questionProjections[id]
	if key == "userQuestions" {
		if p.Questions != nil && seq < p.QuestionsSeq {
			return
		}
		p.Questions, p.QuestionsSeq = append(json.RawMessage(nil), raw...), seq
	} else {
		if p.Inbox != nil && seq < p.InboxSeq {
			return
		}
		p.Inbox, p.InboxSeq = append(json.RawMessage(nil), raw...), seq
		queued := questionQueuedCalls(raw)
		if len(queued) > 0 {
			if p.QueueSeen == nil {
				p.QueueSeen = map[string]uint64{}
			}
			ordinal := questionQueueClock.Add(1)
			for id := range queued {
				p.QueueSeen[id] = ordinal
			}
		}
	}
	p.pruneQueueSeen()
	s.questionProjections[id] = p
}

func (s *Store) applyQuestionControlLocked(raw json.RawMessage) {
	var frame struct {
		Type      string          `json:"type"`
		SessionID string          `json:"sessionId"`
		Key       string          `json:"key"`
		Seq       float64         `json:"seq"`
		Value     json.RawMessage `json:"value"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	switch frame.Type {
	case "projection":
		s.applyQuestionFaceLocked(frame.SessionID, frame.Key, frame.Value, frame.Seq)
	case "baseline":
		var baseline struct {
			Projections map[string]struct {
				AsOfSeq float64         `json:"asOfSeq"`
				Values  json.RawMessage `json:"values"`
			} `json:"projections"`
		}
		if json.Unmarshal(frame.Value, &baseline) != nil {
			return
		}
		for id, p := range baseline.Projections {
			s.applyQuestionValuesForSessionLocked(id, p.Values, p.AsOfSeq)
		}
	}
}
