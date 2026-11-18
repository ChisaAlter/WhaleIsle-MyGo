package main

import (
	"context"
	"encoding/json"
	"errors"
	"whaleisle/internal/errcopy"

	"github.com/egoist/mygo"
	"whaleisle/internal/recovery"
	"whaleisle/internal/store/session"
)

// Bind only after the current local profile's client and shared mux are ready.
func (s *mainState) bindSubmissionRecovery(ctx context.Context, sessions *session.Store) {
	s.mu.Lock()
	store, win := s.resumeStore, s.win
	s.mu.Unlock()
	if sessions == nil {
		return
	}
	owner := questionDraftProfileOwner()
	var raw json.RawMessage
	if store != nil {
		raw = store.Snapshot().Submissions
	}
	err := sessions.BindSubmissionRecovery(ctx, owner, raw, func(record session.SubmissionRecord) error {
		if store == nil {
			return errors.New("submission recovery storage unavailable")
		}
		var invalid error
		err := store.Update(func(state *recovery.State) {
			invalid = applySubmissionRecord(state, owner, record)
		})
		if err != nil {
			return err
		}
		return invalid
	})
	if err != nil {
		sessions.RequireSubmissionRecovery(err)
		sessions.SetLastAction(errcopy.Sentence(err))
		return
	}
	var saved session.SubmissionRecovery
	if json.Unmarshal(raw, &saved) == nil {
		apply := func() {
			s.mu.Lock()
			for _, record := range saved.Records {
				if record.Phase != "confirmed" {
					key := record.DraftKey
					if record.DraftSequence == 0 {
						key = record.SessionID
					}
					s.composerDraftLocked(key).RestoreSubmissionFlight(record)
				}
			}
			s.mu.Unlock()
			s.reconcileComposerSubmissions(sessions)
		}
		if win != nil {
			mygo.RunOnMain(apply)
		} else {
			apply()
		}
	}
}

// The request and its first draft checkpoint commit together. A later UI
// checkpoint or a faster second submission may already own a newer draft.
func applySubmissionRecord(state *recovery.State, owner string, record session.SubmissionRecord) error {
	saved := session.SubmissionRecovery{Version: 1, Owner: owner, Records: make(map[string]session.SubmissionRecord)}
	if len(state.Submissions) > 0 {
		if err := json.Unmarshal(state.Submissions, &saved); err != nil {
			return err
		}
		if saved.Version != 1 || saved.Owner != owner {
			return errors.New("submission recovery owner changed")
		}
	}
	if saved.Records == nil {
		saved.Records = make(map[string]session.SubmissionRecord)
	}
	if prior, ok := saved.Records[record.RequestID]; ok && (prior.Phase == "confirmed" || prior.Phase == "failed") {
		return nil
	}
	if len(record.DraftSnapshot) > 0 {
		var prior struct{ SendSequence uint64 }
		_, exists := state.Drafts[record.DraftKey]
		if !exists || json.Unmarshal(state.Drafts[record.DraftKey], &prior) != nil || prior.SendSequence < record.DraftSequence {
			if state.Drafts == nil {
				state.Drafts = make(map[string]json.RawMessage)
			}
			state.Drafts[record.DraftKey] = append(json.RawMessage(nil), record.DraftSnapshot...)
		}
	}
	if record.Phase == "confirmed" {
		record.Text, record.Attachments = "", nil
	}
	saved.Records[record.RequestID] = record
	raw, err := json.Marshal(saved)
	if err == nil {
		state.Submissions = raw
	}
	return err
}

// Called on the native UI owner. Persistence precedes the published terminal
// outcome, so removing a flight cannot turn a crash into an automatic resend.
func (s *mainState) reconcileComposerSubmissions(sessions *session.Store) {
	if sessions == nil {
		return
	}
	s.mu.Lock()
	if s.sessions != sessions {
		s.mu.Unlock()
		return
	}
	selectID := ""
	settledRequests := make(map[string]bool)
	for key, draft := range s.composerDrafts {
		for _, flight := range draft.PendingSubmissionFlights() {
			if outcome, settled := sessions.SubmissionOutcome(flight.RequestID); settled {
				draft.CompleteDetachedSend(flight.ID, flight.Attachments, outcome)
				settledRequests[flight.RequestID] = true
				if outcome == nil && flight.Exclusive && key == "" && sessions.SelectedSessionID() == "" {
					selectID = flight.SessionID
				}
			}
		}
	}
	if selectID != "" {
		s.composerDrafts[selectID] = s.composerDrafts[""]
		delete(s.composerDrafts, "")
	}
	s.mu.Unlock()
	for id := range settledRequests {
		sessions.AcknowledgeSubmissionOutcome(id)
	}
	if selectID != "" {
		sessions.Select(selectID)
		sessions.RefreshList(s.lifecycleContext())
		sessions.FollowSelected(s.lifecycleContext())
	}
}
