package session

import (
	"context"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func (s *Store) runCreateNoDirectorySession(ctx context.Context, client *dsh.Client, hostGeneration uint64, selectedID string, excludeIDs []string) {
	s.mu.Lock()
	currentHost := s.client == client && s.eventsGeneration == hostGeneration
	s.mu.Unlock()
	if !currentHost {
		return
	}
	id, err := s.CreateNoDirectorySession(ctx, excludeIDs...)
	s.mu.Lock()
	if s.client != client || s.eventsGeneration != hostGeneration {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.lastAction = "no workspace folder: " + err.Error()
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	changed := s.selectedID == selectedID && s.selectedID != id
	if changed {
		s.selectSessionLocked(id)
	}
	s.mu.Unlock()
	if changed {
		s.fire()
		s.refreshSelectedViews()
	}
}

func (s *Store) runCreateSession(ctx context.Context, client *dsh.Client, hostGeneration uint64, cwd, stagedPreset string) {
	req := api.SessionCreateRequest{}
	if cwd != "" {
		req.Cwd = &cwd
	}
	if stagedPreset != "" {
		req.AgentPreset = &stagedPreset
	}
	var out api.SessionCreateValue
	err := client.Call(ctx, "session/create", map[string]any{"request": req}, &out)
	s.mu.Lock()
	if s.client != client || s.eventsGeneration != hostGeneration {
		s.mu.Unlock()
		return
	}
	if err != nil {
		if s.lastAction != err.Error() {
			s.invalidateViewLocked()
		}
		s.lastAction = err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	if stagedPreset != "" && s.stagedPresetLocked(cwd) == stagedPreset {
		delete(s.presetDraft.staged, filepathClean(cwd))
	}
	s.selectSessionLocked(out.SessionId)
	s.lastAction = "session created"
	s.mu.Unlock()
	s.fire()
	s.RefreshList(ctx)
	s.FollowSelected(ctx)
	s.InitializeDefaultModel(ctx)
}

func (s *Store) runForkSession(ctx context.Context, client *dsh.Client, hostGeneration uint64, sessionID string) {
	req := api.SessionForkRequest{SessionId: sessionID}
	var out struct {
		SessionId string `json:"sessionId"`
	}
	err := client.Call(ctx, "session/fork", map[string]any{"request": req}, &out)
	s.mu.Lock()
	if s.client != client || s.eventsGeneration != hostGeneration {
		s.mu.Unlock()
		return
	}
	if err != nil {
		if s.lastAction != err.Error() {
			s.invalidateViewLocked()
		}
		s.lastAction = err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	if out.SessionId != "" {
		s.selectSessionLocked(out.SessionId)
	}
	if s.lastAction != "session forked" {
		s.invalidateViewLocked()
	}
	s.lastAction = "session forked"
	s.mu.Unlock()
	s.fire()
	s.RefreshList(ctx)
	s.FollowSelected(ctx)
}
