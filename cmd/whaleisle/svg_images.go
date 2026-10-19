package main

import (
	"context"
	"fmt"
	"os"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/svgimage"
)

type nativeSVGImages struct {
	window   *mygo.Window
	sessions *session.Store
	host     uint64
	selected string
	owner    *svgimage.Owner
	cancel   context.CancelFunc
}

func (s *mainState) svgImageOwner(window *mygo.Window, sessions *session.Store) *svgimage.Owner {
	if window == nil || sessions == nil {
		return nil
	}
	snapshot := sessions.Snapshot()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quitting || s.svgImagesPaused || s.win != window || s.sessions != sessions {
		return nil
	}
	old := s.svgImages
	if old != nil && old.window == window && old.sessions == sessions && old.host == snapshot.ComposerHostGeneration && old.selected == snapshot.SelectedID {
		return old.owner
	}
	if old != nil {
		old.cancel()
	}
	exe, err := os.Executable()
	if err != nil {
		s.svgImages = nil
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	scope := &nativeSVGImages{window: window, sessions: sessions, host: snapshot.ComposerHostGeneration, selected: snapshot.SelectedID, cancel: cancel}
	scope.owner = &svgimage.Owner{ID: fmt.Sprintf("%p/%p/%d/%s", window, sessions, scope.host, scope.selected), Window: window, Context: ctx, HelperPath: previewprocess.HelperPath(exe)}
	scope.owner.Valid = func() bool {
		if ctx.Err() != nil {
			return false
		}
		s.mu.Lock()
		current := s.svgImages == scope && !s.quitting && !s.svgImagesPaused && s.win == window && s.sessions == sessions
		s.mu.Unlock()
		if !current {
			return false
		}
		latest := sessions.Snapshot()
		return latest.ComposerHostGeneration == scope.host && latest.SelectedID == scope.selected
	}
	s.svgImages = scope
	return scope.owner
}

func (s *mainState) pauseSVGImages() {
	s.mu.Lock()
	s.svgImagesPaused = true
	old := s.svgImages
	s.svgImages = nil
	s.mu.Unlock()
	if old != nil {
		old.cancel()
	}
}

func (s *mainState) resumeSVGImages() {
	s.mu.Lock()
	s.svgImagesPaused = false
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
