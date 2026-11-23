package main

import (
	"context"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/store/session"
)

// nativeMedia is the main window's image lease: one owner per window,
// session store, Host generation and selected session. Changing any of them,
// or hiding, minimising or closing the window, releases it, which drops every
// picture still waiting for a decode worker and cancels the one decoding.
// Decoded pictures stay in the process cache for the next lease.
type nativeMedia struct {
	window   *mygo.Window
	sessions *session.Store
	host     uint64
	selected string
	owner    *media.Owner
	cancel   context.CancelFunc
}

func (s *mainState) mediaOwner(window *mygo.Window, sessions *session.Store) *media.Owner {
	if window == nil || sessions == nil {
		return nil
	}
	snapshot := sessions.SelectionSummary()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quitting || s.mediaPaused || s.win != window || s.sessions != sessions {
		return nil
	}
	old := s.media
	if old != nil && old.window == window && old.sessions == sessions && old.host == snapshot.ComposerHostGeneration && old.selected == snapshot.SelectedID {
		return old.owner
	}
	if old != nil {
		old.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	scope := &nativeMedia{window: window, sessions: sessions, host: snapshot.ComposerHostGeneration, selected: snapshot.SelectedID, cancel: cancel}
	scope.owner = media.Default().NewOwner(ctx, window.Invalidate)
	s.media = scope
	return scope.owner
}

func (s *mainState) pauseMedia() {
	s.mu.Lock()
	s.mediaPaused = true
	old := s.media
	s.media = nil
	s.mu.Unlock()
	if old != nil {
		old.cancel()
	}
}

func (s *mainState) resumeMedia() {
	s.mu.Lock()
	s.mediaPaused = false
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
