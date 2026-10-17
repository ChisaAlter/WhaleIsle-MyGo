package session

import (
	"context"
	"encoding/json"

	"whaleisle/internal/dsh/api"
)

const historyPageMaxMessages = 50

func sessionAddressJSON(sessionID string) json.RawMessage {
	addr, _ := json.Marshal(map[string]any{
		"kind":      "session",
		"sessionId": sessionID,
	})
	return addr
}

func oldestPersistenceSeq(msgs []Message) float64 {
	var min float64
	for _, m := range msgs {
		if m.Seq <= 0 {
			continue
		}
		if min == 0 || m.Seq < min {
			min = m.Seq
		}
	}
	return min
}

func maxPersistenceSeq(msgs []Message) float64 {
	var max float64
	for _, m := range msgs {
		if m.Seq > max {
			max = m.Seq
		}
	}
	return max
}

func (s *Store) resetHistoryPagingLocked() {
	s.historyHasMore = false
	s.historyPageBusy = false
	s.historyPageErr = nil
}

func (s *Store) syncHistoryHasMoreAfterSnapshotLocked() {
	oldest := oldestPersistenceSeq(s.messages)
	s.historyHasMore = oldest > 1
}

func (s *Store) bumpHistoryThroughSeq(seq float64) {
	if seq <= 0 {
		return
	}
	if seq > s.historyThroughSeq {
		s.historyThroughSeq = seq
	}
}

// LoadEarlierHistory fetches the next older page via session/page and prepends
// into the active transcript without disturbing live follow streaming.
func (s *Store) LoadEarlierHistory(ctx context.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.historyPageBusy || !s.historyHasMore || s.client == nil || s.selectedID == "" {
		s.mu.Unlock()
		return
	}
	s.historyPageBusy = true
	s.historyPageErr = nil
	client := s.client
	sessionID := s.selectedID
	through := s.historyThroughSeq
	before := oldestPersistenceSeq(s.messages)
	if through <= 0 {
		through = maxPersistenceSeq(s.messages)
	}
	s.mu.Unlock()
	s.fire()

	if before <= 0 || through <= 0 {
		s.mu.Lock()
		s.historyPageBusy = false
		s.historyHasMore = false
		s.mu.Unlock()
		s.fire()
		return
	}

	go func() {
		maxMsg := float64(historyPageMaxMessages)
		beforeSeq := before
		req := api.SessionPageRequest{
			Address:     sessionAddressJSON(sessionID),
			ThroughSeq:  through,
			BeforeSeq:   &beforeSeq,
			MaxMessages: &maxMsg,
		}
		var out api.SessionPage
		err := client.Call(ctx, "session/page", map[string]any{"request": req}, &out)

		s.mu.Lock()
		s.historyPageBusy = false
		if err != nil {
			s.historyPageErr = err
			s.lastAction = "session/page: " + err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.prependPageRecordsLocked(out.Records)
		s.historyHasMore = out.HasMore
		if len(out.Records) == 0 {
			s.historyHasMore = false
		}
		s.lastAction = ""
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) prependPageRecordsLocked(records []api.SessionPageRecordsItem) {
	if len(records) == 0 {
		return
	}
	seen := make(map[float64]struct{}, len(s.messages))
	for _, m := range s.messages {
		if m.Seq > 0 {
			seen[m.Seq] = struct{}{}
		}
	}
	var batch []Message
	for _, rec := range records {
		if rec.Type != "event" {
			continue
		}
		raw, err := json.Marshal(rec.Event)
		if err != nil {
			continue
		}
		s.bumpHistoryThroughSeq(rec.Event.Seq)
		msg, ok := classifyPersistenceEvent(raw)
		if !ok {
			continue
		}
		if msg.ID == "" {
			msg.ID = newRequestID()
		}
		if msg.Seq > 0 {
			if _, dup := seen[msg.Seq]; dup {
				continue
			}
			seen[msg.Seq] = struct{}{}
		}
		batch = append(batch, msg)
	}
	if len(batch) == 0 {
		return
	}
	s.messages = append(batch, s.messages...)
}
