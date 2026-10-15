package main

import (
	"whaleisle/internal/dsh"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

// syncDialogue only attaches the Supervisor-owned local dsh child. Its dsh home
// is stable across auth-cookie renewal and automatic listen-port fallback; an
// unrelated SSH tunnel must not masquerade as the identity of that child.
func (s *mainState) bindWorkspacePreferences(sessions *session.Store, client *dsh.Client) {
	s.mu.Lock()
	cfg := s.cfgStore
	s.mu.Unlock()
	if cfg != nil {
		sessions.BindWorkspacePreferences(client, "supervised-dsh-home:"+paths.DshHome(), cfg)
	}
}
