package main

import (
	"path/filepath"

	"whaleisle/internal/paths"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
)

func (s *mainState) browserWorkspaceOwner() browserpreviewui.WorkspaceOwner {
	s.mu.Lock()
	cfg, sessions, quitting := s.cfgStore, s.sessions, s.quitting
	s.mu.Unlock()
	if quitting || cfg == nil {
		return browserpreviewui.WorkspaceOwner{}
	}
	// syncDialogue connects the Supervisor-owned Host; this is the same stable
	// namespace used by workspace preferences. Auth cookies, listen-port fallback,
	// and process generations must never become persistent Browser identities.
	// This is the desktop-owned local app home, not the Host workspace cwd.
	home, err := filepath.Abs(paths.DshHome())
	if err != nil {
		return browserpreviewui.WorkspaceOwner{}
	}
	owner := browserpreviewui.WorkspaceOwner{Authority: "supervised-dsh-home:" + home}
	if sessions != nil {
		owner.Cwd, owner.Title = sessions.BrowserWorkspaceScope()
	} else {
		owner.Cwd = cfg.Get().Workspace
	}
	return owner
}
