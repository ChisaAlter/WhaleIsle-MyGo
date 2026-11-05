package interaction

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Reopening republishes a request at the end of its domain's insertion order.
func (s *Store) raisePendingLocked(eventID string) {
	for index, pending := range s.pending {
		if pending.EventID == eventID {
			copy(s.pending[index:], s.pending[index+1:])
			s.pending[len(s.pending)-1] = pending
			return
		}
	}
}

// Unnamed questions have renderer-local keys, not a shared Host call ID. The
// complete question schema correlates the single elected plan-review seat.
// The browser separately checks that seat's real key is still the active one.
func unnamedPlanSourceRequestID(raw json.RawMessage, callID string, request userQuestionsRequest) string {
	if callID != "" || planReviewOf(request.Questions) == nil {
		return ""
	}
	var envelope struct {
		Questions json.RawMessage `json:"questions"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return ""
	}
	var questions []any
	if json.Unmarshal(envelope.Questions, &questions) != nil || questions == nil {
		return ""
	}
	var canonical bytes.Buffer
	encoder := json.NewEncoder(&canonical)
	encoder.SetEscapeHTML(false)
	if encoder.Encode(questions) != nil {
		return ""
	}
	return "questions:" + strings.TrimSuffix(canonical.String(), "\n")
}
