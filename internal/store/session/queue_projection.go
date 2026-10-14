package session

import (
	"encoding/json"
	"strings"
)

// The inbox projection is authoritative for current pending rows. Persistence
// previews alone can be truncated or retain occurrences already consumed.
func (s *Store) applyQueueValuesLocked(values json.RawMessage, seq float64) {
	var data map[string]json.RawMessage
	if json.Unmarshal(values, &data) != nil {
		return
	}
	if inbox, ok := data["inbox"]; ok {
		s.applyQueueInboxLocked(inbox, seq)
	}
}
func (s *Store) applyQueueInboxLocked(raw json.RawMessage, seq float64) {
	if s.queueProjectionSessionID == s.selectedID && s.queueProjectionGeneration == s.followGeneration && seq < s.queueProjectionSeq {
		return
	}
	var inbox struct {
		Next []struct {
			ID      string `json:"id"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			Source struct {
				Kind string `json:"kind"`
			} `json:"source"`
		} `json:"next-turn"`
	}
	if json.Unmarshal(raw, &inbox) != nil || inbox.Next == nil {
		return
	}
	next := make([]PlanQueueItem, 0, len(inbox.Next))
	for _, row := range inbox.Next {
		var parts []string
		for _, part := range row.Content {
			if part.Type == "text" {
				parts = append(parts, part.Text)
			} else {
				parts = append(parts, "["+part.Type+"]")
			}
		}
		next = append(next, PlanQueueItem{ID: row.ID, Preview: truncateRunes(strings.Join(parts, ""), 120), Source: row.Source.Kind})
	}
	s.queueProjectionSessionID = s.selectedID
	s.queueProjectionGeneration = s.followGeneration
	s.queueProjectionSeq = seq
	s.planBoard.Queue = next
}
func (s *Store) applyQueueControlLocked(raw json.RawMessage) {
	var frame struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Type != "projection" || frame.SessionID != s.selectedID || frame.Key != "inbox" {
		return
	}
	s.applyQueueInboxLocked(frame.Value, frame.Seq)
}
