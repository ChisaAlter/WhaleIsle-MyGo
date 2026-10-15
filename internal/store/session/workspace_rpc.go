package session

import (
	"context"
	"encoding/json"
	"fmt"
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
			title := strings.TrimSpace(out.Workspace.Title)
			if title == "" {
				title = strings.TrimSpace(out.Workspace.Path)
			}
			if title == "" {
				title = out.Workspace.WorkspaceId
			}
			s.lastAction = "workspace ready: " + title
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

// InsertWorkspaceBefore reorders workspaces via workspace/insertBefore.
func (s *Store) InsertWorkspaceBefore(ctx context.Context, workspaceID, beforeWorkspaceID string) {
	workspaceID = strings.TrimSpace(workspaceID)
	beforeWorkspaceID = strings.TrimSpace(beforeWorkspaceID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || workspaceID == "" {
		return
	}
	go func() {
		req := api.WorkspaceInsertBeforeRequest{WorkspaceId: workspaceID}
		if beforeWorkspaceID != "" {
			req.BeforeWorkspaceId = &beforeWorkspaceID
		}
		var out api.WorkspaceOrderValue
		err := client.Call(ctx, "workspace/insertBefore", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = "workspace order: " + err.Error()
		} else {
			s.lastAction = fmt.Sprintf("workspace order: %d", len(out.WorkspaceIds))
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// InsertSessionBefore reorders a session within a workspace via workspace/insertSessionBefore.
func (s *Store) InsertSessionBefore(ctx context.Context, workspaceID, sessionID, beforeSessionID string) {
	workspaceID = strings.TrimSpace(workspaceID)
	sessionID = strings.TrimSpace(sessionID)
	beforeSessionID = strings.TrimSpace(beforeSessionID)
	s.mu.Lock()
	client := s.client
	if workspaceID == "" {
		workspaceID = s.workspaceID
	}
	s.mu.Unlock()
	if client == nil || workspaceID == "" || sessionID == "" {
		return
	}
	go func() {
		req := api.WorkspaceInsertSessionBeforeRequest{
			WorkspaceId: workspaceID,
			SessionId:   sessionID,
		}
		if beforeSessionID != "" {
			req.BeforeSessionId = &beforeSessionID
		}
		var out json.RawMessage
		err := client.Call(ctx, "workspace/insertSessionBefore", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = "session order: " + err.Error()
		} else {
			s.lastAction = "session reordered"
		}
		s.mu.Unlock()
		s.fire()
		if err == nil {
			s.RefreshList(ctx)
		}
	}()
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
