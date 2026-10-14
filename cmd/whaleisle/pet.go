package main

import (
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
)

// refreshPetTask uses the same current connection and session roster as chat.
// It does not read arbitrary logs, other workspaces, or infer job completion.
func (s *mainState) refreshPetTask() {
	s.mu.Lock()
	p, sessions := s.desktopPet, s.sessions
	online := s.bootStatus.State == harness.Ready && s.dshMuxReady
	s.mu.Unlock()
	if p == nil {
		return
	}
	snap := session.ViewState{}
	if sessions != nil {
		snap = sessions.Snapshot()
	}
	p.UpdateTask(pet.TaskFromSession(online, snap))
}

func (s *mainState) openPetSession(id string) {
	s.mu.Lock()
	sessions := s.sessions
	online := s.bootStatus.State == harness.Ready && s.dshMuxReady
	if sessions == nil || !online {
		s.mu.Unlock()
		s.refreshPetTask()
		return
	}
	task := pet.TaskFromSession(online, sessions.Snapshot())
	// Recheck the live source: an old pet link cannot open a different workspace,
	// a deleted session, or whichever conversation happens to be selected now.
	if id == "" || task.SessionID != id {
		s.mu.Unlock()
		s.refreshPetTask()
		return
	}
	s.launcherOpen = false
	s.settingsOpen = false
	s.usageOpen = false
	s.scheduleOpen = false
	s.imOpen = false
	s.marketOpen = false
	s.remoteOpen = false
	s.webviewOpen = false
	s.browserPreviewOpen = false
	s.assistantOpen = false
	s.chatPaneTab = "chat"
	win := s.win
	s.mu.Unlock()
	sessions.Select(id)
	s.showMainWindow()
	if win != nil {
		win.Update(func() {})
	}
}
