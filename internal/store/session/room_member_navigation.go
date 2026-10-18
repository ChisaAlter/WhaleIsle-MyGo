package session

import (
	"context"
	"strings"
	"whaleisle/internal/dsh"
)

// SelectObservedRoomMember is admitted only after an explicit runtime-row click
// and a fresh Host-verified root session/list result. It reuses ordinary session
// addresses, history and interaction projections, but pins the selected client,
// mux and previous main selection across admission. No prompt/answer is sent.
func (s *Store) SelectObservedRoomMember(client *dsh.Client, previous, memberSessionID string) bool {
	if memberSessionID == "" || memberSessionID != strings.TrimSpace(memberSessionID) {
		return false
	}
	s.mu.Lock()
	if client == nil || s.client != client || s.mux == nil || s.selectedID != previous {
		s.mu.Unlock()
		return false
	}
	if s.selectedID == memberSessionID {
		s.mu.Unlock()
		return true
	}
	s.selectSessionLocked(memberSessionID)
	mux, generation := s.mux, s.followGeneration
	followCtx, followCancel := context.WithCancel(context.Background())
	s.followCancel = followCancel
	if s.controlCancel != nil {
		s.controlCancel()
	}
	controlCtx, controlCancel := context.WithCancel(context.Background())
	s.controlCancel = controlCancel
	s.controlNote = ""
	s.mu.Unlock()
	s.fire()
	// Captured readers cannot switch to a replacement Host. Attach invalidates
	// follow generation; control frames also require original client+mux identity.
	go s.runFollow(followCtx, mux, memberSessionID, generation)
	go s.runControlWatch(controlCtx, client, mux, memberSessionID)
	return true
}
