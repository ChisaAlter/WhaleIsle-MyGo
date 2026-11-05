package main

import (
	"context"
	"strings"

	"github.com/egoist/mygo"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
)

// The original Tasks page starts a normal workspace conversation. Keep the
// same draft ownership and directory selection as the global New session entry.
func (s *mainState) startScheduleTask(win *mygo.Window, sessions *session.Store) {
	s.openGlobalWorkspaceSession(win, sessions)
}

func scheduleSessionLinkState(snap session.ViewState, id string) string {
	if snap.WorkspaceError != "" || snap.ListErr != "" {
		return "unavailable"
	}
	if !snap.ClientReady || snap.ListLoading || !snap.WorkspaceReady {
		return "loading"
	}
	for _, archived := range snap.ArchivedSessionIDs {
		if archived == id {
			return "archived"
		}
	}
	for _, item := range snap.Items {
		if item.SessionID == id && id != "" {
			return "available"
		}
	}
	return "unavailable"
}

// scheduleSessionLinks mirrors the original task manager's live-list/archive
// gate. Looking at a task or its receipts never restores an archived session.
func scheduleSessionLinks(snap session.ViewState) map[string]string {
	out := map[string]string{}
	if !snap.ClientReady || snap.ListLoading || snap.ListErr != "" || !snap.WorkspaceReady || snap.WorkspaceError != "" {
		return out
	}
	archived := map[string]bool{}
	for _, id := range snap.ArchivedSessionIDs {
		archived[id] = true
	}
	for _, item := range snap.Items {
		if item.SessionID == "" || archived[item.SessionID] {
			continue
		}
		label := strings.TrimSpace(item.Title)
		if label == "" {
			label = item.SessionID
		}
		out[item.SessionID] = label
	}
	return out
}

func (s *mainState) openScheduleSession(row schedstore.Entry) {
	s.mu.Lock()
	sessions, win := s.sessions, s.win
	origin := s.bootStatus.Endpoint.Origin
	s.mu.Unlock()
	if sessions == nil || row.Origin == "" || row.Origin != origin {
		return
	}
	if _, available := scheduleSessionLinks(sessions.Snapshot())[row.SessionID]; !available {
		return
	}
	s.revokePhoneGit()
	sessions.Select(row.SessionID)
	sessions.FollowSelected(context.Background())
	s.mu.Lock()
	s.scheduleOpen = false
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
