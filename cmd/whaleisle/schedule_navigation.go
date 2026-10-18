package main

import (
	"context"
	"strings"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
)

// scheduleSessionLinks mirrors the original task manager's live-list/archive
// gate. Looking at a task or its receipts never restores an archived session.
func scheduleSessionLinks(snap session.ViewState) map[string]string {
	out := map[string]string{}
	if !snap.ClientReady || snap.ListLoading || snap.ListErr != "" {
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
