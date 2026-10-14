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
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var out api.InitializeDefaultResult
		err := client.Call(ctx, "workspace/initializeDefault", map[string]any{}, &out)
		s.mu.Lock()
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
	return s.workspaceID
}

// DeleteWorkspace calls workspace/delete for a workspace id.
func (s *Store) DeleteWorkspace(ctx context.Context, workspaceID string) {
	workspaceID = strings.TrimSpace(workspaceID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || workspaceID == "" {
		return
	}
	go func() {
		req := api.WorkspaceDeleteRequest{WorkspaceId: workspaceID}
		var out api.WorkspaceDeleteValue
		err := client.Call(ctx, "workspace/delete", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = "workspace delete: " + err.Error()
		} else if out.Deleted {
			s.lastAction = "workspace deleted"
			if s.workspaceID == workspaceID {
				s.workspaceID = ""
			}
		} else {
			s.lastAction = "workspace not deleted"
		}
		s.mu.Unlock()
		s.fire()
		if err == nil && out.Deleted {
			s.RefreshList(ctx)
		}
	}()
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

// RenameWorkspace calls workspace/rename for a workspace id.
func (s *Store) RenameWorkspace(ctx context.Context, workspaceID, title string) {
	workspaceID = strings.TrimSpace(workspaceID)
	title = strings.TrimSpace(title)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || workspaceID == "" || title == "" {
		return
	}
	go func() {
		req := api.WorkspaceRenameRequest{WorkspaceId: workspaceID, Title: title}
		var out json.RawMessage
		err := client.Call(ctx, "workspace/rename", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = "workspace rename: " + err.Error()
		} else {
			s.lastAction = "workspace renamed: " + title
		}
		s.mu.Unlock()
		s.fire()
	}()
}
