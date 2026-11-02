package session

import (
	"context"
	"encoding/json"
	"errors"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Match the original ordinary window: at least 50 messages and two turn
// starts, capped by the Host at 500 messages. One click still reads one page.
const historyPageMaxMessages = 500

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
	s.invalidateViewLocked()
	s.resetHistoryNavigationLocked()
	s.historyPageRequest++
	s.historyHasMore = false
	s.historyPageBusy = false
	s.historyPageErr = nil
}

func (s *Store) syncHistoryHasMoreAfterSnapshotLocked() {
	s.invalidateViewLocked()
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
	if s.historyPageBusy || !s.historyHasMore || s.client == nil || s.selectedID == "" || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	s.historyPageRequest++
	requestID := s.historyPageRequest
	s.historyPageBusy = true
	s.historyPageErr = nil
	s.invalidateViewLocked()
	client := s.client
	sessionID := s.selectedID
	generation := s.followGeneration
	through := s.historyThroughSeq
	before := s.historyBeforeSeqLocked()
	if through <= 0 {
		through = maxPersistenceSeq(s.messages)
	}
	s.mu.Unlock()
	s.fire()

	if before <= 0 || through <= 0 {
		s.mu.Lock()
		if !s.historyPageOwnedLocked(client, sessionID, generation, requestID) {
			s.mu.Unlock()
			return
		}
		s.historyPageBusy = false
		if ctx.Err() == nil {
			s.historyHasMore = false
		}
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		return
	}

	go s.runHistoryPage(ctx, client, sessionID, generation, through, before, requestID)
}

func (s *Store) runHistoryPage(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, through, before float64, requestID uint64) {
	maxMsg := float64(historyPageMaxMessages)
	beforeSeq := before
	req := api.SessionPageRequest{
		Address:     sessionAddressJSON(sessionID),
		ThroughSeq:  through,
		BeforeSeq:   &beforeSeq,
		MaxMessages: &maxMsg,
		TurnWindow:  &api.SessionFollowRequestTurnWindow{MinMessages: 50, MinTurns: 2},
	}
	var out api.SessionPage
	err := client.Call(ctx, "session/page", map[string]any{"request": req}, &out)
	if err == nil {
		err = validateHistoryPage(out.Records, through, before)
	}

	s.mu.Lock()
	if !s.historyPageOwnedLocked(client, sessionID, generation, requestID) {
		s.mu.Unlock()
		return
	}
	s.historyPageBusy = false
	s.invalidateViewLocked()
	if ctx.Err() != nil {
		s.mu.Unlock()
		s.fire()
		return
	}
	if err != nil {
		s.historyPageErr = err
		s.lastAction = "session/page: " + err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	// RemoteJournalStream validates the whole ordered page, then discards the
	// portion already owned by the current window before publishing the prepend.
	older := out.Records[:0]
	for _, record := range out.Records {
		if record.Event.Seq < before {
			older = append(older, record)
		}
	}
	s.prependPageRecordsLocked(older)
	s.historyHasMore = out.HasMore
	if len(out.Records) == 0 {
		s.historyHasMore = false
	}
	s.lastAction = ""
	s.mu.Unlock()
	s.fire()
}

// The Host history is dense. Match RemoteJournalStream.prepend's page and
// join checks before publishing anything, rather than presenting a page with
// a missing interval as a successful older-history load.
func validateHistoryPage(records []api.SessionPageRecordsItem, through, before float64) error {
	var acceptedTail float64
	accepted := false
	for i, record := range records {
		seq := record.Event.Seq
		if record.Type != "event" || !historySafeNumber(seq) || seq > through {
			return errors.New("History page crossed its requested cursor")
		}
		if i > 0 && seq != records[i-1].Event.Seq+1 {
			return errors.New("History page contains a missing interval")
		}
		if seq < before {
			acceptedTail, accepted = seq, true
		}
	}
	if accepted && acceptedTail+1 != before {
		return errors.New("History page does not join the current window")
	}
	return nil
}

func (s *Store) prependPageRecordsLocked(records []api.SessionPageRecordsItem) {
	s.invalidateViewLocked()
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
	var turnSeq float64
	opening := false
	for _, rec := range records {
		if rec.Type != "event" {
			continue
		}
		raw, err := json.Marshal(rec.Event)
		if err != nil {
			continue
		}
		if isToolTimingEventType(rec.Event.Type) && rec.Event.Seq > 0 {
			readToolLifecycleMemo(rec.Event.Seq, raw)
		}
		s.captureInspectorEventWithHeadLocked(raw, rec.Event.Type, rec.Event.Seq)
		s.bumpHistoryThroughSeq(rec.Event.Seq)
		s.noteHistoryBaseLocked(rec.Event.Seq)
		s.captureWorkflowBoundaryLocked(raw)
		s.captureTrajectoryBoundaryWithSeqLocked(raw, rec.Event.Seq)
		if rec.Event.Type == "turn/start" {
			turnSeq, opening = rec.Event.Seq, true
		}
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
		annotateEditMessage(&msg, turnSeq, &opening)
		batch = append(batch, msg)
	}
	if len(batch) == 0 {
		return
	}
	s.messages = append(batch, s.messages...)
}

func (s *Store) historyPageOwnedLocked(client *dsh.Client, sessionID string, generation, requestID uint64) bool {
	return s.client == client && s.selectedID == sessionID && s.followGeneration == generation && s.historyPageRequest == requestID
}
