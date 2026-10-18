package session

import "encoding/json"

// QuotaNotice is a live append failure offered to the frame-wide notice host.
// It survives navigation; historical replay never publishes it.
type QuotaNotice struct {
	ID              uint64
	Code, SessionID string
	Turn            float64
}
type quotaNoticeState struct {
	next    uint64
	current QuotaNotice
	held    bool
}

func (s *Store) QuotaNotice() QuotaNotice {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quotaNotice.current
}
func (s *Store) DismissQuotaNotice(id uint64) {
	s.mu.Lock()
	if s.quotaNotice.current.ID == id {
		s.quotaNotice.current = QuotaNotice{}
		s.quotaNotice.held = false
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) HoldQuotaNotice(id uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == 0 || s.quotaNotice.current.ID != id {
		return false
	}
	s.quotaNotice.held = true
	return true
}
func (s *Store) ReleaseQuotaNotice(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quotaNotice.current.ID == id {
		s.quotaNotice.held = false
	}
}
func (s *Store) observeLiveQuotaLocked(raw json.RawMessage) {
	var event struct {
		Type string `json:"type"`
		Data struct {
			Turn   float64 `json:"turn"`
			Source struct {
				Kind string `json:"kind"`
				Edit *struct {
					Turn float64 `json:"turn"`
				} `json:"edit"`
			} `json:"source"`
			Reason struct {
				Kind  string `json:"kind"`
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			} `json:"reason"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &event) != nil {
		return
	}
	state := &s.quotaNotice
	if event.Type == "user/message" && event.Data.Source.Kind == "user" && event.Data.Source.Edit != nil && state.current.SessionID == s.selectedID && state.current.Turn == event.Data.Source.Edit.Turn {
		state.current = QuotaNotice{}
		state.held = false
	}
	code := event.Data.Reason.Error.Code
	if state.held || event.Type != "turn/end" || event.Data.Reason.Kind != "error" || (code != "QUOTA" && code != "ACCOUNT_QUOTA") {
		return
	}
	state.next++
	state.current = QuotaNotice{ID: state.next, Code: code, SessionID: s.selectedID, Turn: event.Data.Turn}
}
