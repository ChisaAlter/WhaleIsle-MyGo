package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

const (
	MessageFeedbackUp   = "up"
	MessageFeedbackDown = "down"
)

// RateMessage records thumbs up/down for one assistant message. When the harness
// exposes messageFeedback/put it is called; otherwise only LastAction is set.
// note is optional (typically a one-line dislike reason). Concurrent submits are
// ignored while FeedbackBusyID is set.
func (s *Store) RateMessage(ctx context.Context, messageID, rating, note string) {
	rating = strings.TrimSpace(rating)
	if rating == "" {
		return
	}
	note = strings.TrimSpace(note)

	s.mu.Lock()
	if s.feedbackBusyID != "" {
		s.mu.Unlock()
		return
	}
	if prev, ok := s.messageRatings[messageID]; ok && prev == rating && note == "" {
		s.mu.Unlock()
		return
	}
	client := s.client
	sessionID := s.selectedID
	s.feedbackBusyID = messageID
	s.mu.Unlock()
	s.fire()

	go func() {
		defer func() {
			s.mu.Lock()
			if s.feedbackBusyID == messageID {
				s.feedbackBusyID = ""
			}
			s.mu.Unlock()
			s.fire()
		}()

		local := func(detail string, saveRating bool) {
			s.mu.Lock()
			s.lastAction = detail
			if saveRating {
				if s.messageRatings == nil {
					s.messageRatings = make(map[string]string)
				}
				s.messageRatings[messageID] = rating
			}
			s.mu.Unlock()
			s.fire()
		}

		if client == nil || sessionID == "" {
			local("feedback "+rating+" (offline)", true)
			return
		}
		if messageID == "" {
			local("feedback "+rating+" (local)", true)
			return
		}
		req := api.MessageFeedbackPutRequest{
			SessionId: sessionID,
			MessageId: messageID,
			Rating:    rating,
			IfVersion: "",
		}
		if note != "" {
			n := note
			req.Note = &n
		}
		var out api.MessageFeedbackPutResult
		err := client.Call(ctx, "messageFeedback/put", map[string]any{"request": req}, &out)
		if err != nil {
			if feedbackEndpointMissing(err) {
				local("feedback "+rating+" (local)", true)
				return
			}
			local("feedback: "+err.Error(), false)
			return
		}
		if note != "" {
			// Best-effort session-level note via sessionFeedback/record (Electron parity).
			cat := rating
			text := note
			var sessOut api.SessionFeedbackRecordResult
			_ = client.Call(ctx, "sessionFeedback/record", map[string]any{
				"request": api.SessionFeedbackRecordRequest{
					SessionId: sessionID,
					Text:      &text,
					Category:  &cat,
				},
			}, &sessOut)
		}
		local("feedback "+rating+" saved", true)
	}()
}

func feedbackEndpointMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "http 404") || strings.Contains(msg, "not found") {
		return true
	}
	var dshErr *dsh.Error
	if errors.As(err, &dshErr) {
		code := strings.ToLower(dshErr.Code)
		return code == "not_found" || code == "method_not_found" || code == "unknown_method"
	}
	return false
}

func (s *Store) clearFeedbackLocked() {
	s.messageRatings = nil
	s.feedbackBusyID = ""
}

// RefreshFeedback loads messageFeedback/list for the selected session into messageRatings.
func (s *Store) RefreshFeedback(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go func() {
		var raw api.MessageFeedbackListResult
		err := client.Call(ctx, "messageFeedback/list", map[string]any{
			"request": map[string]any{"sessionId": sessionID},
		}, &raw)
		if err != nil {
			return
		}
		ratings := parseFeedbackList(raw)
		if len(ratings) == 0 {
			return
		}
		s.mu.Lock()
		if s.selectedID != sessionID {
			s.mu.Unlock()
			return
		}
		if s.messageRatings == nil {
			s.messageRatings = make(map[string]string)
		}
		for id, rating := range ratings {
			s.messageRatings[id] = rating
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ClearMessageFeedback calls messageFeedback/delete and drops the local rating.
func (s *Store) ClearMessageFeedback(ctx context.Context, messageID string) {
	messageID = strings.TrimSpace(messageID)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	if messageID == "" || sessionID == "" {
		s.mu.Unlock()
		return
	}
	delete(s.messageRatings, messageID)
	s.mu.Unlock()
	s.fire()
	if client == nil {
		return
	}
	go func() {
		var out api.MessageFeedbackDeleteResult
		_ = client.Call(ctx, "messageFeedback/delete", map[string]any{
			"request": api.MessageFeedbackDeleteRequest{
				SessionId: sessionID,
				MessageId: messageID,
				IfVersion: "",
			},
		}, &out)
	}()
}

func parseFeedbackList(raw json.RawMessage) map[string]string {
	out := map[string]string{}
	if len(raw) == 0 {
		return out
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		var wrap map[string]json.RawMessage
		if err := json.Unmarshal(raw, &wrap); err != nil {
			return out
		}
		for _, key := range []string{"items", "feedback", "list", "entries"} {
			if chunk, ok := wrap[key]; ok {
				_ = json.Unmarshal(chunk, &arr)
				break
			}
		}
	}
	for _, item := range arr {
		id := jsonRawString(item["messageId"])
		if id == "" {
			id = jsonRawString(item["id"])
		}
		rating := jsonRawString(item["rating"])
		if id != "" && rating != "" {
			out[id] = rating
		}
	}
	return out
}

func jsonRawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	return ""
}
