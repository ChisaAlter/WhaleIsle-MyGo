package session

import (
	"bytes"
	"encoding/json"
)

// QuestionProjectionState preserves both authoritative faces needed for late replies.
// nil means not loaded; it must never be confused with an authoritative empty inbox.
type QuestionProjectionState struct {
	Questions    json.RawMessage
	Inbox        json.RawMessage
	QuestionsSeq float64
	InboxSeq     float64
}

func (s *Store) questionProjectionLocked() QuestionProjectionState {
	p := s.questionProjections[s.selectedID]
	p.Questions = append(json.RawMessage(nil), p.Questions...)
	p.Inbox = append(json.RawMessage(nil), p.Inbox...)
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
	}
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
