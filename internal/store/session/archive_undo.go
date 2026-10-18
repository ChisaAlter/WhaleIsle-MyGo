package session

import (
	"context"
	"slices"
	"time"
	"whaleisle/internal/dsh"
)

const archiveUndoLifetime = 8 * time.Second

type ArchiveUndoNotice struct {
	Token            uint64
	SessionID, Title string
	Expires          time.Time
	Announce         bool
}
type archiveUndoState struct {
	notice     ArchiveUndoNotice
	client     *dsh.Client
	generation uint64
	announced  bool
}

func (s *Store) offerArchiveUndoLocked(client *dsh.Client, id, title string, now time.Time) {
	s.archiveUndoToken++
	s.archiveUndo = &archiveUndoState{notice: ArchiveUndoNotice{Token: s.archiveUndoToken, SessionID: id, Title: title, Expires: now.Add(archiveUndoLifetime)}, client: client, generation: s.eventsGeneration}
}
func (s *Store) archiveUndoCurrentLocked(now time.Time) bool {
	u := s.archiveUndo
	return u != nil && s.client == u.client && s.eventsGeneration == u.generation && now.Before(u.notice.Expires)
}

// The notice is runtime-only and announced once. The UI closes its toast at the
// absolute deadline too, even if the native toast timer pauses while hovered.
func (s *Store) ArchiveUndoNotice() *ArchiveUndoNotice {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.archiveUndoCurrentLocked(time.Now()) {
		s.archiveUndo = nil
		return nil
	}
	out := s.archiveUndo.notice
	out.Announce = !s.archiveUndo.announced
	s.archiveUndo.announced = true
	return &out
}
func (s *Store) UndoArchive(token uint64) {
	s.mu.Lock()
	if !s.archiveUndoCurrentLocked(time.Now()) || s.archiveUndo.notice.Token != token || !slices.Contains(s.archivedSessionIDs, s.archiveUndo.notice.SessionID) {
		s.mu.Unlock()
		return
	}
	u := *s.archiveUndo
	s.archiveUndo = nil // claim once before dispatch; repeated clicks are inert
	s.mu.Unlock()
	s.fire()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	go func() { defer cancel(); s.runUnarchiveSession(ctx, u.client, u.generation, u.notice.SessionID) }()
}
