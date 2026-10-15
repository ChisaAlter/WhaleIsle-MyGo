package main

import (
	"context"

	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/harness"
)

// The compact card is a second native surface on Whale's persistent session.
// It never selects or rewrites the main composer until the explicit ↗ action.
func (s *mainState) openPetQuickChat() {
	s.mu.Lock()
	p, sessions, cfg, win, host := s.desktopPet, s.sessions, s.cfgStore, s.win, s.bootStatus
	ready := !s.quitting && s.dshMuxReady && host.State == harness.Ready
	s.mu.Unlock()
	if p == nil || !p.ChatAllowed() {
		return
	}
	unavailable := ""
	if cfg == nil || !cfg.Get().WhaleAssistantEnabled {
		unavailable = "请先在助手设置启用鲸鱼娘 / Whale assistant is disabled"
	} else if !ready || sessions == nil || !sessions.Snapshot().ClientReady || win == nil || win.IsDestroyed() {
		unavailable = "Host 未连接。恢复后重新点聊聊。 / Host offline; reopen after connecting."
	}
	current := func() bool {
		s.mu.Lock()
		now := s.bootStatus
		valid := !s.quitting && s.win == win && s.desktopPet == p && s.sessions == sessions && s.cfgStore == cfg && s.dshMuxReady && now.State == harness.Ready && now.Generation == host.Generation && now.Endpoint.Origin == host.Endpoint.Origin && now.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		return valid && cfg != nil && cfg.Get().WhaleAssistantEnabled && p.ChatAllowed() && sessions != nil && sessions.Snapshot().ClientReady && win != nil && !win.IsDestroyed()
	}
	p.OpenQuickChat(pet.QuickChatOptions{
		Origin: host.Endpoint.Origin, Cookie: host.Endpoint.Cookie, Unavailable: unavailable,
		Current: current,
		OpenMain: func(id string) {
			// The card just re-read pet/state for this exact identity. This last
			// UI-thread guard rejects a Host/window replacement before navigation.
			if id == "" || !current() {
				return
			}
			s.activatePetConversation(sessions, id)
			sessions.RefreshList(context.Background())
		},
	})
}
