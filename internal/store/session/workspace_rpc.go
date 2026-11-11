package session

import (
	"context"
	"os"
	"strings"

	"whaleisle/internal/dsh/api"
)

// InitializeDefaultWorkspace calls workspace/initializeDefault (Electron first-run workspace).
func (s *Store) InitializeDefaultWorkspace(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	generation, token, cwd, selected := s.eventsGeneration, s.workspaceNavigationToken, s.cwd, s.selectedID
	initial := &s.initialWorkspaceSession
	if client == nil || ctx == nil || ctx.Err() != nil || (initial.armed && (initial.completed || initial.pending && initial.requestHost == generation)) {
		s.mu.Unlock()
		return
	}
	if !initial.armed {
		initial.armed, initial.eligible = true, selected == "" && s.workspaceNavigation.view.Mode == ""
		initial.navigation, initial.follow, initial.cwd = token, s.followGeneration, cwd
	}
	// A restored registry is already initialized. Keep its selected directory;
	// only a genuinely empty registry needs the first-run RPC.
	if s.hasOrdinarySessionsLocked() || s.workspaceReady && len(s.workspaceMemberships) != 0 {
		initial.completed = true
		initial.eligible = false
		for _, row := range s.workspaceRowsLocked() {
			if hostPathKey(row.Path) == hostPathKey(cwd) {
				s.workspaceID = row.WorkspaceID
				break
			}
		}
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		s.RefreshList(ctx)
		return
	}
	initial.pending, initial.requestHost = true, generation
	binding := initial.binding
	s.mu.Unlock()
	go func() {
		var out api.InitializeDefaultResult
		err := client.Call(ctx, "workspace/initializeDefault", map[string]any{}, &out)
		var createdRow WorkspaceRow
		if err == nil && strings.TrimSpace(out.Workspace.WorkspaceId) == "" && strings.TrimSpace(cwd) != "" {
			s.mu.Lock()
			canCreate := s.client == client && s.eventsGeneration == generation &&
				s.initialWorkspaceSession.binding == binding && s.initialWorkspaceSession.requestHost == generation &&
				s.workspaceNavigationToken == token && s.cwd == cwd && s.selectedID == selected &&
				len(s.workspaceMemberships) == 0 && !s.hasOrdinarySessionsLocked()
			s.mu.Unlock()
			if canCreate {
				if info, statErr := os.Stat(cwd); statErr == nil && info.IsDir() {
					var created api.WorkspaceCreateValue
					if errCreate := client.Call(ctx, "workspace/create", map[string]any{"request": map[string]any{"path": cwd}}, &created); errCreate == nil {
						createdRow = workspaceRowFromCreate(created.Workspace)
						out.Workspace = created.Workspace
					}
				}
			}
		}
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation || s.initialWorkspaceSession.binding != binding || s.initialWorkspaceSession.requestHost != generation {
			s.mu.Unlock()
			return
		}
		s.initialWorkspaceSession.pending = false
		s.initialWorkspaceSession.completed = true
		if s.workspaceNavigationToken != token || s.cwd != cwd || s.selectedID != selected {
			s.mu.Unlock()
			return
		}
		if ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = "workspace init: " + err.Error()
		} else {
			if createdRow.WorkspaceID != "" && createdRow.Path != "" {
				s.putWorkspaceRowLocked(createdRow)
			}
			id := strings.TrimSpace(out.Workspace.WorkspaceId)
			path := strings.TrimSpace(out.Workspace.Path)
			if id != "" && path != "" {
				s.initialWorkspaceSession.defaultID = id
				s.initialWorkspaceSession.defaultCwd = path
			}
			if selected == "" && (cwd == "" || hostPathKey(cwd) == hostPathKey(out.Workspace.Path)) {
				// A successful initializeDefault is silent: the status line belongs to
				// connection and failure facts, not to a routine workspace handshake.
				if id != "" {
					s.workspaceID = id
				}
				if path != "" {
					s.cwd = path
				}
				initial := &s.initialWorkspaceSession
				if initial.eligible && initial.navigation == token && initial.follow == s.followGeneration && initial.cwd == cwd && id != "" && path != "" {
					initial.accepted = true
					initial.workspaceID, initial.cwd = id, path
				}
			} else {
				s.reconcileInitialDefaultWorkspaceLocked()
			}
		}
		s.invalidateViewLocked()
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

// InsertSessionBefore changes browser-local order without editing Host membership.
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
