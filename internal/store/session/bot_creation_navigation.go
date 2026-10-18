package session

import (
	"context"
	"strings"
	"whaleisle/internal/dsh"
)

// BotCreationOpenTarget captures the original Host, mux and selection generation
// before asynchronous canonical opening. An away-and-back selection or Host swap
// cannot revive it, even when the visible session ID happens to match again.
type BotCreationOpenTarget struct {
	store      *Store
	client     *dsh.Client
	mux        *dsh.Mux
	previous   string
	generation uint64
	consumed   bool
}

func (s *Store) CaptureBotCreationOpen(client *dsh.Client) *BotCreationOpenTarget {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client == nil || s.client != client || s.mux == nil {
		return nil
	}
	return &BotCreationOpenTarget{store: s, client: client, mux: s.mux, previous: s.selectedID, generation: s.followGeneration}
}
func (t *BotCreationOpenTarget) currentLocked() bool {
	return t != nil && !t.consumed && t.store.client == t.client && t.store.mux == t.mux && t.store.selectedID == t.previous && t.store.followGeneration == t.generation
}
func (t *BotCreationOpenTarget) Current() bool {
	if t == nil || t.store == nil {
		return false
	}
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	return t.currentLocked()
}
func (t *BotCreationOpenTarget) Select(sessionID string) bool {
	if t == nil || t.store == nil || sessionID == "" || sessionID != strings.TrimSpace(sessionID) {
		return false
	}
	s := t.store
	s.mu.Lock()
	if !t.currentLocked() {
		s.mu.Unlock()
		return false
	}
	t.consumed = true
	if s.selectedID == sessionID {
		s.invalidateFollowLocked()
	} else {
		s.selectSessionLocked(sessionID)
	}
	generation := s.followGeneration
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
	// These observers only use the captured original identities. Never call a
	// helper that reads the current selection/client after this admission.
	go s.runFollow(followCtx, t.mux, sessionID, generation)
	go s.runControlWatch(controlCtx, t.client, t.mux, sessionID)
	return true
}
