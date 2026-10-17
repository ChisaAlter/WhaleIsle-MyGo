package main

import "whaleisle/internal/ui/frame"

func (s *mainState) navigationFileDropModel(m frame.Model) frame.Model {
	s.mu.Lock()
	blocked := s.settingsOpen || s.launcherOpen || s.marketOpen || s.scheduleOpen || s.assistantOpen || s.usageOpen || s.imOpen || s.remoteOpen || s.webviewOpen || s.browserPreviewOpen || s.sessions != m.Store
	if blocked || m.Store == nil {
		s.mu.Unlock()
		return m
	}
	snap := m.Store.Snapshot()
	draft := s.composerDraftLocked(snap.SelectedID)
	win := s.win
	s.mu.Unlock()
	m.Draft, m.ComposerState = &draft.Text, draft
	m.OnAttachComposerFiles = func(paths []string) {
		s.mu.Lock()
		valid := s.sessions == m.Store && s.composerDrafts[snap.SelectedID] == draft && !s.settingsOpen && !s.launcherOpen && !s.marketOpen && !s.scheduleOpen && !s.assistantOpen && !s.usageOpen && !s.imOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen
		s.mu.Unlock()
		live := m.Store.Snapshot()
		if !valid || live.SelectedID != snap.SelectedID || live.ComposerHostGeneration != snap.ComposerHostGeneration {
			return
		}
		s.addComposerFiles(win, m.Store, snap.SelectedID, draft, paths)
	}
	return m
}
