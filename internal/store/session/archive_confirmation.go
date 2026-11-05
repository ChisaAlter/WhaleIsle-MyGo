package session

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// ArchiveActivity is the Host's authoritative list of work that must stop.
type ArchiveActivity struct {
	Kind  string                `json:"kind"`
	Items []ArchiveActivityItem `json:"items,omitempty"`
}
type ArchiveActivityItem struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type ArchiveConfirmation struct {
	Token     uint64
	SessionID string
	Title     string
	Activity  []ArchiveActivity
	Busy      bool
	Error     string
}

type archiveOperation struct {
	confirmation ArchiveConfirmation
	waiting      bool
	cancel       context.CancelFunc
}

func (s *Store) archiveConfirmationLocked() *ArchiveConfirmation {
	if !s.archive.waiting {
		return nil
	}
	out := s.archive.confirmation
	out.Activity = append([]ArchiveActivity(nil), out.Activity...)
	for i := range out.Activity {
		out.Activity[i].Items = append([]ArchiveActivityItem(nil), out.Activity[i].Items...)
	}
	return &out
}

// A new selection/Host invalidates the whole dialog, including late refusals.
func (s *Store) invalidateArchiveLocked() {
	s.invalidateViewLocked()
	if s.archive.cancel != nil {
		s.archive.cancel()
	}
	s.archiveToken++
	s.archive = archiveOperation{}
}

func (s *Store) beginArchive(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	if s.client == nil || s.archive.confirmation.SessionID != "" {
		s.mu.Unlock()
		return
	}
	s.archiveToken++
	token, client := s.archiveToken, s.client
	title := sessionID
	for _, item := range s.items {
		if item.SessionID == sessionID && item.Title != "" {
			title = item.Title
			break
		}
	}
	callCtx, cancel := context.WithCancel(ctx)
	s.archive = archiveOperation{confirmation: ArchiveConfirmation{Token: token, SessionID: sessionID, Title: title, Busy: true}, cancel: cancel}
	s.mu.Unlock()
	s.fire()
	go s.runArchive(callCtx, client, token, sessionID, false)
}

// ConfirmArchive can stop work only for the exact Host refusal shown to the user.
func (s *Store) ConfirmArchive(ctx context.Context, token uint64) {
	s.mu.Lock()
	if !s.archive.waiting || s.archive.confirmation.Token != token || s.archive.confirmation.Busy || ctx.Err() != nil || s.client == nil {
		s.mu.Unlock()
		return
	}
	sessionID, client := s.archive.confirmation.SessionID, s.client
	if s.archive.cancel != nil {
		s.archive.cancel()
	}
	callCtx, cancel := context.WithCancel(ctx)
	s.archive.cancel = cancel
	s.archive.confirmation.Busy, s.archive.confirmation.Error = true, ""
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	go s.runArchive(callCtx, client, token, sessionID, true)
}

func (s *Store) CancelArchive(token uint64) {
	s.mu.Lock()
	if !s.archive.waiting || s.archive.confirmation.Token != token || s.archive.confirmation.Busy {
		s.mu.Unlock()
		return
	}
	s.invalidateArchiveLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) runArchive(ctx context.Context, client *dsh.Client, token uint64, sessionID string, stopActivity bool) {
	req := api.WorkspaceArchiveSessionRequest{SessionId: sessionID}
	if stopActivity {
		req.StopActivity = &stopActivity
	}
	var out api.WorkspaceArchiveValue
	err := client.Call(ctx, "workspace/archiveSession", map[string]any{"request": req}, &out)
	s.mu.Lock()
	if s.client != client || s.archiveToken != token || s.archive.confirmation.SessionID != sessionID {
		s.mu.Unlock()
		return
	}
	s.archive.confirmation.Busy = false
	s.invalidateViewLocked()
	if err != nil {
		activity, refused := archiveRefusal(err, sessionID)
		if refused {
			s.archive.waiting = true
			s.archive.confirmation.Activity = activity
		}
		if s.archive.waiting {
			if stopActivity || !refused {
				s.archive.confirmation.Error = err.Error()
			}
		} else {
			s.lastAction = err.Error()
			s.invalidateArchiveLocked()
		}
		s.mu.Unlock()
		s.fire()
		return
	}
	title := s.archive.confirmation.Title
	s.invalidateArchiveLocked()
	// Do not clear a newer selection because the original row was selected.
	if s.selectedID == sessionID {
		s.selectSessionLocked("")
	}
	s.archivedSessionIDs = append([]string(nil), out.ArchivedSessionIds...)
	s.lastAction = "session archived"
	if slices.Contains(out.ArchivedSessionIds, sessionID) {
		s.offerArchiveUndoLocked(client, sessionID, title, stopActivity, time.Now())
	}
	s.mu.Unlock()
	s.fire()
	// The operation context has ended; refresh on a fresh context.
	s.RefreshList(context.Background())
}

func archiveRefusal(err error, sessionID string) ([]ArchiveActivity, bool) {
	var failure *dsh.Error
	if !errors.As(err, &failure) || failure.Code != "workspace/session-active" {
		return nil, false
	}
	var details struct {
		SessionID string            `json:"sessionId"`
		Activity  []ArchiveActivity `json:"activity"`
	}
	if json.Unmarshal(failure.Details, &details) != nil || details.SessionID != sessionID || len(details.Activity) == 0 {
		return nil, false
	}
	for _, activity := range details.Activity {
		if strings.TrimSpace(activity.Kind) == "" {
			return nil, false
		}
	}
	return details.Activity, true
}
