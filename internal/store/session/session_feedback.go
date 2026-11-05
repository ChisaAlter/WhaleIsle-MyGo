package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"whaleisle/internal/dsh/api"
)

// SessionFeedbackView is a local draft; opening or editing never records feedback.
type SessionFeedbackView struct {
	SessionID, Text, Category, Error string
	Open, Busy                       bool
}

func (s *Store) SessionFeedback() SessionFeedbackView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionFeedback
}
func (s *Store) invalidateSessionFeedbackLocked() {
	s.invalidateViewLocked()
	s.sessionFeedbackEpoch++
	s.sessionFeedback = SessionFeedbackView{}
}
func (s *Store) OpenSessionFeedback(id string) bool {
	s.mu.Lock()
	if id == "" {
		id = s.selectedID
	}
	if id == "" || s.sessionFeedback.Busy {
		s.mu.Unlock()
		return false
	}
	s.sessionFeedbackEpoch++
	s.sessionFeedback = SessionFeedbackView{SessionID: id, Open: true}
	s.mu.Unlock()
	s.fire()
	return true
}
func (s *Store) EditSessionFeedback(text, category string) {
	s.mu.Lock()
	if s.sessionFeedback.Open && !s.sessionFeedback.Busy {
		s.sessionFeedback.Text = text
		s.sessionFeedback.Category = category
		s.sessionFeedback.Error = ""
	}
	s.mu.Unlock()
}
func (s *Store) CancelSessionFeedback() {
	s.mu.Lock()
	// Dismissing revokes the editor, not the already admitted record RPC. Its
	// late acknowledgement cannot reopen this draft or replay the submission.
	s.invalidateSessionFeedbackLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SubmitSessionFeedback(ctx context.Context) {
	s.mu.Lock()
	v := s.sessionFeedback
	client := s.client
	epoch := s.sessionFeedbackEpoch
	if !v.Open || v.Busy || v.SessionID == "" {
		s.mu.Unlock()
		return
	}
	if client == nil {
		s.sessionFeedback.Error = "Host is not connected"
		s.mu.Unlock()
		s.fire()
		return
	}
	switch v.Category {
	case "", "task-result", "instruction-following", "product-interaction", "service-stability", "resource-cost", "security-privacy-permission", "other":
	default:
		s.sessionFeedback.Error = "Unknown feedback category"
		s.mu.Unlock()
		s.fire()
		return
	}
	req := api.SessionFeedbackRecordRequest{SessionId: v.SessionID}
	if text := strings.TrimSpace(v.Text); text != "" {
		req.Text = &text
	}
	if v.Category != "" {
		req.Category = &v.Category
	}
	s.sessionFeedback.Busy = true
	s.sessionFeedback.Error = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		var raw json.RawMessage
		err := client.Call(ctx, "sessionFeedback/record", map[string]any{"request": req}, &raw)
		if err == nil {
			err = checkSessionFeedbackAck(raw)
		}
		s.mu.Lock()
		if s.client != client || s.sessionFeedbackEpoch != epoch {
			s.mu.Unlock()
			return
		}
		s.sessionFeedback.Busy = false
		if err != nil {
			s.sessionFeedback.Error = err.Error()
		} else {
			s.invalidateSessionFeedbackLocked()
			s.lastAction = "Session feedback recorded"
		}
		s.mu.Unlock()
		s.fire()
	}()
}
func checkSessionFeedbackAck(raw json.RawMessage) error {
	var r struct {
		OK    *bool `json:"ok"`
		Value struct {
			Recorded bool `json:"recorded"`
		} `json:"value"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &r) != nil || r.OK == nil {
		return fmt.Errorf("feedback: incompatible Host response")
	}
	if !*r.OK {
		return fmt.Errorf("feedback: %s", r.Error.Code)
	}
	if !r.Value.Recorded {
		return fmt.Errorf("feedback: missing recorded acknowledgement")
	}
	return nil
}
