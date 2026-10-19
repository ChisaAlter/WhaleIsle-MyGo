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
	var receipts struct {
		NextTurn []composerInboxMessage `json:"next-turn"`
		NextStep []composerInboxMessage `json:"next-step"`
	}
	if json.Unmarshal(raw, &receipts) == nil {
		s.observeComposerInboxInsertionsLocked("next-turn", receipts.NextTurn, 0, seq)
		s.observeComposerInboxInsertionsLocked("next-step", receipts.NextStep, 0, seq)
	}
	var inbox struct {
		Next []struct {
			ID      string              `json:"id"`
			Content []queueContentBlock `json:"content"`
			Source  struct {
				Kind  string `json:"kind"`
				RPCID string `json:"rpcId"`
			} `json:"source"`
		} `json:"next-turn"`
	}
	if json.Unmarshal(raw, &inbox) != nil || inbox.Next == nil {
		return
	}
	next := make([]PlanQueueItem, 0, len(inbox.Next))
	for _, row := range inbox.Next {
		requestID := ""
		if row.Source.Kind == "user" {
			requestID = row.Source.RPCID
			s.observeComposerRequestLocked(requestID, seq, true)
		}
		if s.hideComposerQueueLocked(requestID) {
			continue
		}
		var parts []string
		item := PlanQueueItem{ID: row.ID, Source: row.Source.Kind, RequestID: requestID, ContentKnown: true, TextOnly: true}
		for _, part := range row.Content {
			if part.Type == "text" {
				parts = append(parts, part.Text)
				continue
			}
			item.TextOnly = false
			if part.Type == "file" || part.Type == "image" {
				if part.Attachment != nil && part.Attachment.AttachmentID != "" {
					attachment := *part.Attachment
					attachment.Kind = part.Type
					item.Attachments = append(item.Attachments, attachment)
				} else {
					parts = append(parts, "["+part.Type+"]")
				}
			} else {
				parts = append(parts, "["+part.Type+"]")
			}
		}
		item.Preview = truncateRunes(strings.Join(parts, " "), 120)
		next = append(next, item)
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
