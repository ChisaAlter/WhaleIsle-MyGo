package main

import (
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
)

func (s *mainState) navigationFileDropModel(m frame.Model) frame.Model {
	s.mu.Lock()
	blocked := s.settingsOpen || s.launcherOpen || s.marketOpen || s.scheduleOpen || s.assistantOpen || s.usageOpen || s.imOpen || s.remoteOpen || s.webviewOpen || s.browserPreviewOpen || s.sessions != m.Store
	if blocked || m.Store == nil {
		s.mu.Unlock()
		return m
	}
	// Only the selection is needed here. Snapshot() builds turn usage (JSON
	// unmarshalling), workspaces and file lists; calling it on every frame kept
	// the UI thread decoding JSON instead of handling input.
	selectedID := m.Store.SelectedSessionID()
	draft := s.composerDraftLocked(selectedID)
	win := s.win
	s.mu.Unlock()
	m.Draft, m.ComposerState = &draft.Text, draft
	m.OnAttachComposerFiles = func(paths []string) {
		s.mu.Lock()
		valid := s.sessions == m.Store && s.composerDrafts[selectedID] == draft && !s.settingsOpen && !s.launcherOpen && !s.marketOpen && !s.scheduleOpen && !s.assistantOpen && !s.usageOpen && !s.imOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen
		s.mu.Unlock()
		if !valid || m.Store.SelectedSessionID() != selectedID {
			return
		}
		s.addComposerFiles(win, m.Store, selectedID, draft, paths)
	}
	return m
}

// sessionFrameViews keeps one read-only frame handle per store the render
// passes have touched. The handles outlive a pass on purpose: each Snapshot
// revalidates against the store, and keeping them lets an unchanged revision be
// cloned once instead of once per pass. A store replacement cannot close another
// store's handle or make the new store reuse the old store's view; the handles
// of stores that are no longer active are released at the end of the pass.
type sessionFrameViews map[*session.Store]*session.FrameSnapshot

func (views sessionFrameViews) view(store *session.Store) *session.FrameSnapshot {
	if store == nil {
		return nil
	}
	frame := views[store]
	if frame == nil {
		frame = store.BeginFrame()
		views[store] = frame
	}
	return frame
}

func (views sessionFrameViews) snapshot(store *session.Store) session.ViewState {
	return views.view(store).Snapshot()
}

func (views sessionFrameViews) end() {
	for _, frame := range views {
		frame.EndFrame()
	}
}

func (s *mainState) beginSessionFrameViews() sessionFrameViews {
	if s.frameViews == nil {
		s.frameViews = make(sessionFrameViews)
	}
	return s.frameViews
}

// endSessionFrameViews releases the handles of every store but the active one.
// The active store's handle is kept for the next pass.
func (s *mainState) endSessionFrameViews(views sessionFrameViews) {
	s.mu.Lock()
	active := s.sessions
	s.mu.Unlock()
	for store, frame := range views {
		if store != active {
			frame.EndFrame()
			delete(views, store)
		}
	}
}
