package main

import (
	"context"
	"errors"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/workspacepicker"
)

// Saving config is the last fallible step before Store commits its selection.
// Do not acquire mainState.mu here: this callback runs under the Store lock.
func (s *mainState) persistRegisteredWorkspace(path string) error {
	if s.cfgStore == nil {
		return errors.New("Workspace settings are unavailable")
	}
	return s.cfgStore.Update(func(c *config.Config) { c.Workspace = path })
}
func (s *mainState) workspaceSelectionCommitted(win *mygo.Window, sessions *session.Store, path string) {
	s.mu.Lock()
	if sessions != nil && sessions.WorkspaceCwd() == path && sessions.Snapshot().SelectedID == "" {
		if home := s.composerDrafts[""]; home != nil && !home.Sending && home.Preparing == 0 {
			home.TargetID = ""
		}
	}
	s.mu.Unlock()
	s.revokePhoneGit()
	if win != nil {
		win.Update(func() {})
	}
}
func (s *mainState) workspaceNavigationView(c *ui.Context, win *mygo.Window, sessions *session.Store, themeID string, systemDark bool, onNoDirectory func()) {
	workspacepicker.View(c, workspacepicker.Model{Store: sessions, ThemeID: themeID, SystemDark: systemDark, OnNoDirectory: onNoDirectory,
		OnChoose: func(token uint64, id string) {
			if sessions.ChooseRegisteredWorkspace(token, id, s.persistRegisteredWorkspace) {
				s.workspaceSelectionCommitted(win, sessions, sessions.WorkspaceCwd())
			}
		},
		OnRegister: func(token uint64, path string) {
			sessions.RegisterWorkspace(context.Background(), token, path, s.persistRegisteredWorkspace, func(row session.WorkspaceRow) { s.workspaceSelectionCommitted(win, sessions, row.Path) })
		},
	})
}
