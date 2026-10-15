package main

import (
	"context"

	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/petdirect"
)

// The compact card is a second native surface on Whale's persistent session.
// It never selects or rewrites the main composer until the explicit ↗ action.
func (s *mainState) openPetQuickChat() {
	s.mu.Lock()
	p, sessions, cfg, win, host, owner := s.desktopPet, s.sessions, s.cfgStore, s.win, s.bootStatus, s.sup
	muxReady := s.dshMuxReady
	ready := !s.quitting && muxReady && host.State == harness.Ready
	s.mu.Unlock()
	if p == nil || !p.ChatAllowed() {
		return
	}
	unavailable := ""
	userData, home := paths.UserDataDir(), paths.DshHome()
	whaleEnabled := cfg != nil && cfg.Get().WhaleAssistantEnabled
	clientReady := sessions != nil && sessions.Snapshot().ClientReady
	if cfg == nil || !cfg.Get().WhaleAssistantEnabled {
		unavailable = "请先在助手设置启用鲸鱼娘 / Whale assistant is disabled"
	} else if !ready || sessions == nil || !sessions.Snapshot().ClientReady || win == nil || win.IsDestroyed() {
		unavailable = "Host 未连接。恢复后重新点聊聊。 / Host offline; reopen after connecting."
	}
	// This decision is made before any chat submission. No result/error handler
	// changes routes, and credentials are not read until explicit API Send.
	var direct *petdirect.Client
	if unavailable != "" && cfg != nil && win != nil && !win.IsDestroyed() {
		direct = newPetDirectChat(cfg, userData, home)
	}
	current := func() bool {
		s.mu.Lock()
		now := s.bootStatus
		valid := !s.quitting && s.sup == owner && s.win == win && s.desktopPet == p && s.sessions == sessions && s.cfgStore == cfg && s.dshMuxReady == muxReady && now.State == host.State && now.Generation == host.Generation && now.Endpoint.Origin == host.Endpoint.Origin && now.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		if !valid || cfg == nil || cfg.Get().WhaleAssistantEnabled != whaleEnabled || !p.ChatAllowed() || win == nil || win.IsDestroyed() || paths.UserDataDir() != userData || paths.DshHome() != home {
			return false
		}
		nowClientReady := sessions != nil && sessions.Snapshot().ClientReady
		if direct != nil {
			return nowClientReady == clientReady
		}
		return whaleEnabled && ready && nowClientReady
	}
	p.OpenQuickChat(pet.QuickChatOptions{
		Origin: host.Endpoint.Origin, Cookie: host.Endpoint.Cookie, Unavailable: unavailable,
		Current: current, Direct: direct,
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
