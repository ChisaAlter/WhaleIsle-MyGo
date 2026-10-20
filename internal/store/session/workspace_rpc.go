package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
)

// InitializeDefaultWorkspace calls workspace/initializeDefault (Electron first-run workspace).
func (s *Store) InitializeDefaultWorkspace(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	generation, token, cwd, selected := s.eventsGeneration, s.workspaceNavigationToken, s.cwd, s.selectedID
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var out api.InitializeDefaultResult
		err := client.Call(ctx, "workspace/initializeDefault", map[string]any{}, &out)
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation || s.workspaceNavigationToken != token || s.cwd != cwd || s.selectedID != selected {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = "workspace init: " + err.Error()
		} else {
			// A successful initializeDefault is silent: the status line belongs to
			// connection and failure facts, not to a routine workspace handshake.
			if id := strings.TrimSpace(out.Workspace.WorkspaceId); id != "" {
				s.workspaceID = id
			}
			if path := strings.TrimSpace(out.Workspace.Path); path != "" {
				s.cwd = path
			}
		}
		s.mu.Unlock()
		s.fire()
		if err == nil {
			s.RefreshList(ctx)
		}
	}()
}

// WorkspaceID returns the last workspace/initializeDefault id when known.
func (s *Store) WorkspaceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ws, ok := s.workspaceMemberships[s.workspaceID]; ok && ws.Path == s.cwd {
		return ws.WorkspaceID
	}
	for _, ws := range s.workspaceMemberships {
		if ws.Path == s.cwd {
			return ws.WorkspaceID
		}
	}
	return ""
}

// DeleteWorkspace opens confirmation; it never removes a registration directly.
func (s *Store) DeleteWorkspace(ctx context.Context, workspaceID string) {
	if ctx.Err() == nil {
		s.BeginWorkspaceRemoval(workspaceID)
	}
}

// InsertWorkspaceBefore reorders the real Host registry and projects its receipt.
func (s *Store) InsertWorkspaceBefore(ctx context.Context, workspaceID, beforeWorkspaceID string) {
	s.reorderWorkspace(ctx, workspaceID, beforeWorkspaceID)
}

// InsertSessionBefore moves a member within its owning Host workspace.
func (s *Store) InsertSessionBefore(ctx context.Context, workspaceID, sessionID, beforeSessionID string) {
	s.reorderWorkspaceSession(ctx, workspaceID, sessionID, beforeSessionID)
}

// RenameWorkspace opens an editable, retryable rename draft.
func (s *Store) RenameWorkspace(ctx context.Context, workspaceID, title string) {
	if ctx.Err() != nil {
		return
	}
	s.BeginWorkspaceRename(workspaceID)
	s.mu.Lock()
	if s.workspaceNavigation.view.Mode == "rename" && s.workspaceNavigation.view.TargetID == workspaceID && strings.TrimSpace(title) != "" {
		s.workspaceNavigation.view.Draft = title
	}
	s.mu.Unlock()
	s.fire()
}
