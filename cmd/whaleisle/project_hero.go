package main

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/projects"
)

type projectDirectoryCheck struct {
	reason string
}

func projectDirectoryKey(path string) string { return strings.ToLower(filepath.Clean(path)) }

// The Hero option follows the selected Session's cwd, never the application's
// default workspace. The native source action authorizes only this same owner.
func (s *projectController) bindProjectHero(win *mygo.Window, m *projects.Model, sessions *session.Store, client *dsh.Client, snapshot session.ViewState) {
	path := snapshot.SelectedCwd
	if !m.Listing.Available || client == nil || path == "" || projectDirectoryKey(path) == projectDirectoryKey(snapshot.ScratchCwd) {
		return
	}
	m.HeroWorkspace = path
	for _, project := range m.Listing.Projects {
		if project.Lifecycle != "archived" && projectDirectoryKey(project.CanonicalWorkingDirectory) == projectDirectoryKey(path) {
			m.HeroExisting = true
			break
		}
	}
	key := projectDirectoryKey(path)
	m.HeroDirectoryReason = s.projectDirectoryCheckReason(win, client, path)
	s.mu.Lock()
	check := s.projectFlow.directoryChecks[key]
	m.HeroError = s.projectFlow.createError
	s.mu.Unlock()
	m.OnCreateFromDraft = func() {
		now := sessions.SelectionSummary()
		if now.SelectedID != snapshot.SelectedID || projectDirectoryKey(now.SelectedCwd) != key {
			return
		}
		s.dshMu.Lock()
		currentClient := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		current := check != nil && s.projectFlow.client == client && s.projectFlow.directoryChecks[key] == check && check.reason == ""
		s.mu.Unlock()
		if currentClient == client && current {
			s.mainState.createProjectWithDraft(win, path)
		}
	}
}

// projectDirectoryCheckReason shares the published asynchronous preflight
// between Hero and registered Workspace rows; no render waits for an RPC.
func (s *projectController) projectDirectoryCheckReason(win *mygo.Window, client *dsh.Client, path string) string {
	if client == nil || path == "" {
		return ""
	}
	key := projectDirectoryKey(path)
	s.mu.Lock()
	f := s.projectFlow
	if f.client != client {
		s.mu.Unlock()
		return ""
	}
	if f.directoryChecks == nil {
		f.directoryChecks = map[string]*projectDirectoryCheck{}
	}
	check := f.directoryChecks[key]
	initial := check == nil
	if initial {
		check = &projectDirectoryCheck{}
		f.directoryChecks[key] = check
	}
	reason := check.reason
	s.mu.Unlock()
	if initial && win != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
			result, err := rpc.CheckDirectory(ctx, path)
			win.Update(func() {
				s.mu.Lock()
				defer s.mu.Unlock()
				if s.projectFlow.client != client || s.projectFlow.directoryChecks[key] != check {
					return
				}
				// As in the installed client, a failed preflight leaves creation
				// available; the real creation boundary still enforces the rule.
				if err == nil && !result.Allowed {
					check.reason = result.Reason
					if check.reason == "" {
						check.reason = "unavailable"
					}
				}
			})
		}()
	}
	return reason
}
