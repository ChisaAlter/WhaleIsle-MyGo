package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func (s *Store) submissionWatchCurrentLocked(sessionID string, w *submissionWatch) bool {
	return w.ctx.Err() == nil && s.client == w.client && s.mux == w.mux && s.composerHostGeneration == w.epoch && s.submissionWatches[sessionID] == w
}

// One lightweight journal observer exists per session with unresolved sends.
// It never folds messages into the selected transcript and never requests the
// assistant token stream. Leaving a view cannot revoke its submission owner.
func (s *Store) runSubmissionWatch(sessionID string, w *submissionWatch) {
	completed := false
	defer w.cancel()
	defer func() {
		s.mu.Lock()
		if s.submissionWatches[sessionID] == w {
			delete(s.submissionWatches, sessionID)
			if completed {
				s.startSubmissionWatchesLocked()
			}
		}
		s.mu.Unlock()
	}()
	backoff := 100 * time.Millisecond
	for {
		err := s.followSubmissionOutcomes(sessionID, w)
		if err == nil || w.ctx.Err() != nil {
			completed = err == nil
			return
		}
		s.mu.Lock()
		current := s.submissionWatchCurrentLocked(sessionID, w)
		for _, p := range s.composerPending {
			if current && p.SessionID == sessionID && p.target.currentLocked() {
				p.Unconfirmed = true
				p.recovered = true // A reconnect must search any missed durable page.
			}
		}
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		if !current || (!errors.Is(err, dsh.ErrCarrierLost) && !errors.Is(err, dsh.ErrStreamEnd)) {
			return
		}
		// Reopen only an interrupted existing confirmation stream, matching the
		// transport recovery policy used by the ordinary selected-session follow.
		timer := time.NewTimer(backoff)
		select {
		case <-w.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff = min(2*time.Second, backoff*2)
	}
}

func (s *Store) followSubmissionOutcomes(sessionID string, w *submissionWatch) error {
	if err := w.mux.Dial(w.ctx); err != nil {
		return err
	}
	stream, err := w.mux.Open(w.ctx, "session/follow", map[string]any{"args": map[string]any{"request": map[string]any{
		// Host declares assistantStream as optional literal true. Omitting it
		// disables token frames; false is rejected by the transport schema.
		"address": sessionAddressJSON(sessionID), "maxMessages": historyPageMaxMessages,
	}}})
	if err != nil {
		return err
	}
	defer stream.Cancel()
	type received struct {
		raw json.RawMessage
		err error
	}
	frames := make(chan received, 1)
	recvCtx, stopRecv := context.WithCancel(w.ctx)
	defer stopRecv()
	go func() {
		for {
			raw, err := stream.Recv(recvCtx)
			select {
			case frames <- received{raw, err}:
			case <-recvCtx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	owned := map[string]*composerPending{}
	var cursor float64
	cursorKnown := false
	for {
		s.mu.Lock()
		if !s.submissionWatchCurrentLocked(sessionID, w) {
			s.mu.Unlock()
			return context.Canceled
		}
		for id, p := range s.submissionTracked {
			if p.SessionID == sessionID && p.target.currentLocked() {
				copy := *p
				owned[id] = &copy
			}
		}
		s.mu.Unlock()
		for id, p := range owned {
			s.mu.Lock()
			r := p.target.retirements[id]
			settled := r != nil && r.settled
			var outcome error
			if settled {
				outcome = r.err
			}
			s.mu.Unlock()
			if settled && !errors.Is(outcome, ErrSubmissionUnconfirmed) {
				if err := s.persistSubmissionOutcome(p, outcome); err != nil {
					return err
				}
				delete(owned, id)
			}
		}
		if len(owned) == 0 {
			return nil
		}
		var raw json.RawMessage
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		case <-w.wake:
			continue
		case frame := <-frames:
			raw, err = frame.raw, frame.err
		}
		if err != nil {
			return err
		}
		parsed := parseFollowFrame(raw)
		if parsed.kind == "event" && parsed.eventOK {
			var head struct {
				Seq float64 `json:"seq"`
			}
			if json.Unmarshal(parsed.eventRaw, &head) == nil && cursorKnown {
				if head.Seq <= cursor {
					continue
				}
				if head.Seq != cursor+1 {
					return dsh.ErrStreamEnd
				}
				cursor = head.Seq
			}
		}
		if parsed.kind == "snapshot" && parsed.snapOK {
			if parsed.snap.Cursor != nil {
				cursor, cursorKnown = *parsed.snap.Cursor, true
			}
			// A restart can find the request outside the current visible history
			// page. Scan the bounded Host pages back through this exact cursor.
			for _, rec := range parsed.snap.Records {
				if rec.Type == "event" && rec.head.Type == "user/message" {
					s.observeSubmissionEvent(sessionID, w, rec.Event)
				}
			}
			if parsed.snap.Projections != nil {
				s.observeSubmissionInbox(sessionID, w, parsed.snap.Projections.Values, parsed.snap.Projections.AsOfSeq)
			}
			if err := s.readSubmissionHistory(sessionID, w, parsed.snap, owned); err != nil {
				return err
			}
		} else if parsed.kind == "event" && parsed.eventOK {
			s.observeSubmissionEvent(sessionID, w, parsed.eventRaw)
		}
		for id, p := range owned {
			s.mu.Lock()
			r := p.target.retirements[id]
			settled := r != nil && r.settled
			var outcome error
			if settled {
				outcome = r.err
			}
			s.mu.Unlock()
			if !settled || errors.Is(outcome, ErrSubmissionUnconfirmed) {
				continue
			}
			if err := s.persistSubmissionOutcome(p, outcome); err != nil {
				return err
			}
			delete(owned, id)
		}
	}
}

func (s *Store) observeSubmissionEvent(sessionID string, w *submissionWatch, raw json.RawMessage) {
	s.mu.Lock()
	if s.submissionWatchCurrentLocked(sessionID, w) {
		s.observeComposerEventForSessionLocked(sessionID, raw)
	}
	s.mu.Unlock()
}

func (s *Store) observeSubmissionInbox(sessionID string, w *submissionWatch, values json.RawMessage, seq float64) {
	var projections struct {
		Inbox map[string][]composerInboxMessage `json:"inbox"`
	}
	if json.Unmarshal(values, &projections) != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.submissionWatchCurrentLocked(sessionID, w) {
		return
	}
	for target, messages := range projections.Inbox {
		s.observeComposerInboxInsertionsForSessionLocked(sessionID, target, messages, 0, seq)
		if target == "next-turn" {
			for _, msg := range messages {
				if msg.Source.Kind == "user" {
					s.observeComposerRequestForSessionLocked(sessionID, msg.Source.RPCID, seq, true)
				}
			}
		}
	}
}

func (s *Store) readSubmissionHistory(sessionID string, w *submissionWatch, snap followSnapshotFrame, owned map[string]*composerPending) error {
	if snap.Cursor == nil || snap.HasMore == nil || !*snap.HasMore || len(snap.Records) == 0 {
		return nil
	}
	before := snap.Records[0].seq
	for before > 0 {
		s.mu.Lock()
		pending := false
		for id := range owned {
			pending = pending || s.composerPending[id] != nil && owned[id].recovered
		}
		current := s.submissionWatchCurrentLocked(sessionID, w)
		s.mu.Unlock()
		if !pending || !current {
			return nil
		}
		var page api.SessionEventsPage
		req := api.SessionPageRequest{Address: sessionAddressJSON(sessionID), ThroughSeq: snap.Cursor, BeforeSeq: &before}
		if err := w.client.Call(w.ctx, "session/page", map[string]any{"request": req}, &page); err != nil {
			return err
		}
		for _, rec := range page.Records {
			// An old inbox insertion is not current queue authority. Only a
			// durable user/message can confirm an older journal page.
			if rec.Type == "event" && rec.Event.Type == "user/message" {
				raw, _ := json.Marshal(rec.Event)
				s.observeSubmissionEvent(sessionID, w, raw)
			}
		}
		if !page.HasMore || len(page.Records) == 0 {
			return nil
		}
		next := page.Records[0].Event.Seq
		if next <= 0 || next >= before {
			return errors.New("submission history did not make backward progress")
		}
		before = next
	}
	return nil
}

func (s *Store) persistSubmissionOutcome(p *composerPending, outcome error) error {
	s.mu.Lock()
	writer := s.submissionPersistence
	s.mu.Unlock()
	record := SubmissionRecord{RequestID: p.RequestID, SessionID: p.SessionID, Text: p.Text, Placement: p.Placement,
		Phase: "confirmed", Attachments: append([]ComposerAttachment(nil), p.Attachments...)}
	if p.recoveryText != nil {
		record.Text = *p.recoveryText
	}
	record.DraftKey, record.DraftSequence = p.draftKey, p.draftSequence
	if outcome != nil {
		record.Phase, record.Error = "failed", outcome.Error()
	}
	if writer != nil {
		writer.mu.Lock()
		if writer.terminal[record.RequestID] {
			writer.mu.Unlock()
			return nil
		}
		var err error
		if writer.write == nil {
			err = errors.New("submission recovery storage unavailable")
		} else {
			err = writer.write(record)
		}
		if err == nil {
			writer.terminal[record.RequestID] = true
		}
		if err != nil {
			writer.mu.Unlock()
			return fmt.Errorf("save submission outcome: %w", err)
		}
	}
	s.mu.Lock()
	s.noteRecoveredOutcomeLocked(record)
	delete(s.submissionTracked, record.RequestID)
	delete(p.target.dispatched, record.RequestID)
	if watch := s.submissionWatches[p.SessionID]; watch != nil {
		select {
		case watch.wake <- struct{}{}:
		default:
		}
	}
	s.mu.Unlock()
	if writer != nil {
		writer.mu.Unlock()
	}
	s.fire()
	return nil
}
