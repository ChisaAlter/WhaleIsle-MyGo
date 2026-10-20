package main

import (
	"context"
	"fmt"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// refreshPetTask uses the same current connection and session roster as chat.
// The separate notification reader uses only the running Host's owned data home;
// selected-session status remains derived from the authenticated chat snapshot.
func (s *mainState) refreshPetTask() {
	s.syncPetEvents()
	s.syncPetPersonality()
	s.dshMu.Lock()
	careClient := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	p, sessions, careStore := s.desktopPet, s.sessions, s.cfgStore
	careGeneration := s.bootStatus.Generation
	careOwner := fmt.Sprintf("%p/%p", s.sup, careClient)
	online := s.bootStatus.State == harness.Ready && s.dshMuxReady
	s.mu.Unlock()
	if p == nil {
		return
	}
	p.SetCareRuntime(careOwner, careGeneration)
	if careStore != nil {
		growthOwner := fmt.Sprintf("%s/%q/%q", careOwner, careStore.Get().NodeBin, paths.DshHome())
		p.SetGrowthRuntime(growthOwner, careGeneration)
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
	s.mu.Unlock()
	s.activatePetConversation(sessions, id)
}

// The callers validate either the displayed task identity or the current
// authenticated Whale ensure result before changing the visible conversation.
func (s *mainState) activatePetConversation(sessions *session.Store, id string) {
	s.mu.Lock()
	s.launcherOpen = false
	s.settingsOpen = false
	s.usageOpen = false
	s.scheduleOpen = false
	s.imOpen = false
	s.marketOpen = false
	s.remoteOpen = false
	s.webviewOpen = false
	s.browserPreviewOpen = false
	s.assistantConversationGeneration++
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

// openPetChat is an explicit entry into Whale's own persistent conversation,
// distinct from openPetSession (which returns to the displayed task).
func (s *mainState) openPetChat() {
	s.mu.Lock()
	p, sessions, cfg, win, host := s.desktopPet, s.sessions, s.cfgStore, s.win, s.bootStatus
	ready := !s.quitting && s.dshMuxReady && host.State == harness.Ready
	s.mu.Unlock()
	if p == nil {
		return
	}
	ctx, finish := p.StartChat()
	if ctx == nil {
		return
	}
	if cfg == nil || !cfg.Get().WhaleAssistantEnabled {
		finish(i18n.T(i18n.KeyCmdWhaleAssistantIsDisabled))
		return
	}
	if !ready || sessions == nil || !sessions.Snapshot().ClientReady || win == nil {
		finish(i18n.T(i18n.KeyCmdHostOffline2))
		return
	}
	selected := sessions.Snapshot().SelectedID
	go func() {
		id, err := pet.EnsureChatSession(ctx, host.Endpoint.Origin, host.Endpoint.Cookie)
		win.Update(func() {
			s.mu.Lock()
			now := s.bootStatus
			current := !s.quitting && s.win == win && s.desktopPet == p && s.sessions == sessions && s.cfgStore == cfg && s.dshMuxReady && now.State == harness.Ready && now.Generation == host.Generation && now.Endpoint.Origin == host.Endpoint.Origin && now.Endpoint.Cookie == host.Endpoint.Cookie
			s.mu.Unlock()
			if !current || !cfg.Get().WhaleAssistantEnabled {
				finish(i18n.T(i18n.KeyCmdConnectionChanged))
				return
			}
			if sessions.Snapshot().SelectedID != selected {
				finish(i18n.T(i18n.KeyCmdConversationChanged))
				return
			}
			if err != nil {
				// Host bodies/errors may contain URLs or private data. Do not publish them
				// to the local renderer or silently substitute the current task session.
				finish(i18n.T(i18n.KeyCmdPetChatUnavailable))
				return
			}
			if !finish(i18n.T(i18n.KeyCmdPetChatOpened)) {
				return
			}
			s.activatePetConversation(sessions, id)
			sessions.RefreshList(context.Background())
		})
	}()
}
