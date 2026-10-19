package main

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/settings"
)

type settingsSkillHubState struct {
	open, searching, installing bool
	confirm                     *settings.SkillHubHit
	err                         string
	source                      *session.Store
	target                      session.SkillTarget
	epoch                       uint64
	cancel                      context.CancelFunc
}

func (s *mainState) invalidateSettingsSkillHubLocked() {
	if s.settingsSkillHub.cancel != nil {
		s.settingsSkillHub.cancel()
	}
	s.settingsSkillHub = settingsSkillHubState{epoch: s.settingsSkillHub.epoch + 1}
	s.settingsSkillHubHits = nil
	s.settingsSkillHubTarget = session.SkillTarget{}
}

func (s *mainState) settingsSkillHubCurrentLocked() bool {
	hub := &s.settingsSkillHub
	if hub.source == nil || s.sessions != hub.source {
		return false
	}
	target, err := hub.source.SkillInventoryTarget(s.settingsSkillRoot)
	return err == nil && target == hub.target
}

func (s *mainState) settingsSkillHubAccess() *settings.SkillHubAccess {
	s.mu.Lock()
	defer s.mu.Unlock()
	hub := &s.settingsSkillHub
	if hub.open && hub.source != nil && !s.settingsSkillHubScopeCurrentLocked() {
		s.invalidateSettingsSkillHubLocked()
		hub = &s.settingsSkillHub
	}
	var confirm *settings.SkillHubHit
	if hub.confirm != nil {
		copy := *hub.confirm
		confirm = &copy
	}
	return &settings.SkillHubAccess{
		Open: hub.open, Searching: hub.searching, Installing: hub.installing,
		Confirming: confirm, Error: hub.err,
		OnOpen: s.openSettingsSkillHub, OnClose: s.closeSettingsSkillHub,
		OnCancelInstall: s.cancelSettingsSkillHubInstall, OnConfirmInstall: s.confirmSettingsSkillHubInstall,
	}
}

func (s *mainState) settingsSkillHubScopeCurrentLocked() bool {
	hub := &s.settingsSkillHub
	if hub.source == nil || hub.source != s.sessions {
		return false
	}
	current, err := hub.source.SkillInventoryTarget(session.SkillRootUser)
	return err == nil && hub.target.SameScope(current)
}

func (s *mainState) updateSettingsSkillHubUI() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) openSettingsSkillHub() {
	s.mu.Lock()
	if s.settingsSkillHub.searching || s.settingsSkillHub.installing {
		s.mu.Unlock()
		return
	}
	hub := &s.settingsSkillHub
	hub.open, hub.err, hub.confirm, hub.source = true, "", nil, s.sessions
	if hub.source == nil {
		hub.err = "Runtime disconnected"
	} else {
		var err error
		hub.target, err = hub.source.SkillInventoryTarget(s.settingsSkillRoot)
		if err != nil {
			hub.err = err.Error()
			// Keep the selector usable if a former project target has no cwd.
			hub.target, _ = hub.source.SkillInventoryTarget(session.SkillRootUser)
		}
		if s.settingsSkillHubTarget != hub.target {
			s.settingsSkillHubHits = nil
		}
	}
	s.mu.Unlock()
	s.updateSettingsSkillHubUI()
}

func (s *mainState) closeSettingsSkillHub() {
	s.mu.Lock()
	if !s.settingsSkillHub.searching && !s.settingsSkillHub.installing {
		s.settingsSkillHub.open, s.settingsSkillHub.err, s.settingsSkillHub.confirm = false, "", nil
	}
	s.mu.Unlock()
	s.updateSettingsSkillHubUI()
}

func (s *mainState) selectSettingsSkillHubRoot(root string) {
	s.mu.Lock()
	if s.settingsSkillHub.searching || s.settingsSkillHub.installing {
		s.mu.Unlock()
		return
	}
	if s.sessions == nil {
		s.mu.Unlock()
		return
	}
	target, err := s.sessions.SkillInventoryTarget(root)
	if err != nil {
		s.settingsSkillHub.err = err.Error()
	} else if target != s.settingsSkillHub.target || s.settingsSkillRoot != target.Root {
		open := s.settingsSkillHub.open
		s.invalidateSettingsSkillHubLocked()
		s.settingsSkillRoot = target.Root
		s.settingsSkillHub.open, s.settingsSkillHub.source, s.settingsSkillHub.target = open, s.sessions, target
	}
	s.mu.Unlock()
	s.updateSettingsSkillHubUI()
}

func (s *mainState) searchSettingsSkillHub(query string) {
	query = strings.TrimSpace(query)
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	hub := &s.settingsSkillHub
	if query == "" || client == nil || !hub.open || hub.searching || hub.installing || !s.settingsSkillHubCurrentLocked() || !hub.target.MatchesClient(client) {
		s.mu.Unlock()
		return
	}
	hub.searching, hub.err, hub.confirm = true, "", nil
	hub.epoch++
	epoch, target := hub.epoch, hub.target
	ctx, cancel := context.WithCancel(context.Background())
	hub.cancel = cancel
	s.mu.Unlock()
	s.updateSettingsSkillHubUI()
	go func() {
		limit := float64(15)
		req := api.SkillHubSearchRequest{Query: query, Root: target.Root, Limit: &limit}
		if target.Cwd != "" {
			req.Cwd = &target.Cwd
		}
		if target.SessionID != "" {
			req.SessionId = &target.SessionID
		}
		var out api.SearchHubResult
		err := client.Call(ctx, "skillInventory/searchHub", map[string]any{"request": req}, &out)
		cancel()
		s.mu.Lock()
		if s.settingsSkillHub.epoch != epoch {
			s.mu.Unlock()
			return
		}
		if !s.settingsSkillHubCurrentLocked() {
			s.invalidateSettingsSkillHubLocked()
		} else {
			s.settingsSkillHub.searching, s.settingsSkillHub.cancel = false, nil
			if err != nil {
				s.settingsSkillHub.err = err.Error()
			} else {
				hits := make([]settings.SkillHubHit, 0, len(out))
				for _, item := range out {
					status := ""
					if item.InstallationStatus != nil {
						status = *item.InstallationStatus
					}
					hits = append(hits, settings.SkillHubHit{SkillName: item.SkillName, Repo: item.Repo, Path: item.Path, Description: item.Description, Stars: item.Stars, Status: status})
				}
				s.settingsSkillHubHits, s.settingsSkillHubTarget = hits, target
			}
		}
		s.mu.Unlock()
		s.updateSettingsSkillHubUI()
	}()
}

// Selecting a result only opens the original Hub review; it does not install.
func (s *mainState) requestSettingsSkillHubInstall(repo, path, name string) {
	s.mu.Lock()
	hub := &s.settingsSkillHub
	if !hub.open || hub.searching || hub.installing || !s.settingsSkillHubCurrentLocked() || s.settingsSkillHubTarget != hub.target {
		s.mu.Unlock()
		return
	}
	for _, hit := range s.settingsSkillHubHits {
		if hit.Repo == repo && hit.Path == path && hit.SkillName == name && repo != "" && path != "" && name != "" && hit.Status != "installed" && hit.Status != "conflict" {
			copy := hit
			hub.confirm, hub.err = &copy, ""
			break
		}
	}
	s.mu.Unlock()
	s.updateSettingsSkillHubUI()
}

func (s *mainState) cancelSettingsSkillHubInstall() {
	s.mu.Lock()
	if !s.settingsSkillHub.installing {
		s.settingsSkillHub.confirm, s.settingsSkillHub.err = nil, ""
	}
	s.mu.Unlock()
	s.updateSettingsSkillHubUI()
}

func (s *mainState) confirmSettingsSkillHubInstall() {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	hub := &s.settingsSkillHub
	if client == nil || !hub.open || hub.confirm == nil || hub.searching || hub.installing || !s.settingsSkillHubCurrentLocked() || !hub.target.MatchesClient(client) || s.settingsSkillHubTarget != hub.target {
		s.mu.Unlock()
		return
	}
	hit, target, source := *hub.confirm, hub.target, hub.source
	hub.installing, hub.err = true, ""
	hub.epoch++
	epoch := hub.epoch
	ctx, cancel := context.WithCancel(context.Background())
	hub.cancel = cancel
	s.mu.Unlock()
	s.updateSettingsSkillHubUI()
	go func() {
		req := api.SkillHubInstallRequest{Repo: hit.Repo, Path: hit.Path, SkillName: hit.SkillName, Root: target.Root}
		if target.Cwd != "" {
			req.Cwd = &target.Cwd
		}
		if target.SessionID != "" {
			req.SessionId = &target.SessionID
		}
		var out api.SkillHubInstallResult
		err := client.Call(ctx, "skillInventory/installHub", map[string]any{"request": req}, &out)
		cancel()
		message := ""
		if err == nil {
			message, err = session.SkillHubInstallMessage(out.Status, out.SkillName, out.Directory)
		}
		s.mu.Lock()
		if s.settingsSkillHub.epoch != epoch {
			s.mu.Unlock()
			return
		}
		current := s.settingsSkillHubCurrentLocked()
		if !current {
			s.invalidateSettingsSkillHubLocked()
		} else {
			s.settingsSkillHub.installing, s.settingsSkillHub.cancel = false, nil
			if err != nil {
				s.settingsSkillHub.err = err.Error()
			} else {
				s.settingsSkillHub.confirm = nil
				s.settingsSkillsLastAction = message
				for i := range s.settingsSkillHubHits {
					row := &s.settingsSkillHubHits[i]
					if row.Repo == hit.Repo && row.Path == hit.Path && row.SkillName == hit.SkillName {
						row.Status = "installed"
					}
				}
			}
		}
		s.mu.Unlock()
		if current && err == nil {
			source.RefreshSkills(context.Background())
		}
		s.updateSettingsSkillHubUI()
	}()
}
