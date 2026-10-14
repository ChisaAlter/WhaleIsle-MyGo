package main

import (
	"context"
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
		finish("请先在助手设置启用鲸鱼娘 / Whale assistant is disabled")
		return
	}
	if !ready || sessions == nil || !sessions.Snapshot().ClientReady || win == nil {
		finish("Host 未连接，请稍后重试 / Host offline")
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
				finish("Host 或插件已变化，请重试 / Connection changed")
				return
			}
			if sessions.Snapshot().SelectedID != selected {
				finish("已切换会话，请重新点击 / Conversation changed")
				return
			}
			if err != nil {
				// Host bodies/errors may contain URLs or private data. Do not publish them
				// to the local renderer or silently substitute the current task session.
				finish("宠物会话不可用，请检查鲸鱼娘插件 / Pet chat unavailable")
				return
			}
			if !finish("已打开宠物会话 / Pet chat opened") {
				return
			}
			s.activatePetConversation(sessions, id)
			sessions.RefreshList(context.Background())
		})
	}()
}
