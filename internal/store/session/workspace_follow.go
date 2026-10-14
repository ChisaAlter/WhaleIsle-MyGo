package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// FollowWorkspace opens workspace/follow until ctx ends or Detach cancels it.
func (s *Store) FollowWorkspace(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	if mux == nil {
		s.mu.Unlock()
		return
	}
	if s.workspaceFollowCancel != nil {
		s.workspaceFollowCancel()
		s.workspaceFollowCancel = nil
	}
	fctx, cancel := context.WithCancel(ctx)
	s.workspaceFollowCancel = cancel
	s.mu.Unlock()

	go s.runWorkspaceFollow(fctx, mux)
}

func (s *Store) runWorkspaceFollow(ctx context.Context, mux *dsh.Mux) {
	defer func() {
		s.mu.Lock()
		if s.mux == mux && ctx.Err() == nil {
			s.workspaceErr = errors.New("workspace follow unavailable; reconnect and try again")
			s.signalWorkspaceLocked()
		}
		s.mu.Unlock()
		s.fire()
	}()
	if err := mux.Dial(ctx); err != nil {
		s.mu.Lock()
		s.lastAction = "workspace follow dial: " + err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	stream, err := mux.Open(ctx, "workspace/follow", map[string]any{"args": map[string]any{}})
	if err != nil {
		// Superseded follow (Detach / re-attach) cancels in-flight Open; not a user-facing fault.
		msg := err.Error()
		if errors.Is(err, context.Canceled) || strings.Contains(msg, "context canceled") ||
			strings.Contains(msg, "use of closed network connection") {
			return
		}
		s.mu.Lock()
		s.lastAction = "workspace follow open: " + msg
		s.mu.Unlock()
		s.fire()
		return
	}
	defer stream.Cancel()

	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, dsh.ErrStreamEnd) || errors.Is(err, context.Canceled) {
				return
			}
			s.mu.Lock()
			s.lastAction = "workspace follow: " + err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.applyWorkspaceFollowFrame(raw)
	}
}

func (s *Store) applyWorkspaceFollowFrame(raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return
	}
	switch head.Type {
	case "baseline":
		s.applyWorkspaceBaseline(raw)
	case "upsert", "remove":
		var frame struct {
			Workspace   workspaceMembership `json:"workspace"`
			WorkspaceID string              `json:"workspaceId"`
		}
		if json.Unmarshal(raw, &frame) != nil {
			return
		}
		s.mu.Lock()
		if s.workspaceMemberships == nil {
			s.workspaceMemberships = map[string]workspaceMembership{}
		}
		if head.Type == "remove" {
			delete(s.workspaceMemberships, frame.WorkspaceID)
		} else if frame.Workspace.WorkspaceID != "" {
			s.workspaceMemberships[frame.Workspace.WorkspaceID] = frame.Workspace
		}
		s.mu.Unlock()
		s.fire()
	case "pinned":
		s.applyWorkspacePinned(raw)
	case "archived":
		s.applyWorkspaceArchived(raw)
	}
}

func (s *Store) applyWorkspaceBaseline(raw json.RawMessage) {
	var frame struct {
		Value *struct {
			ScratchCwd         string                `json:"scratchCwd"`
			Items              []workspaceMembership `json:"items"`
			PinnedSessionIds   []string              `json:"pinnedSessionIds"`
			ArchivedSessionIds []string              `json:"archivedSessionIds"`
		} `json:"value"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Value == nil || frame.Value.PinnedSessionIds == nil || frame.Value.ArchivedSessionIds == nil {
		return
	}
	s.mu.Lock()
	s.scratchCwd = strings.TrimSpace(frame.Value.ScratchCwd)
	s.workspaceReady = true
	s.workspaceErr = nil
	s.workspaceMemberships = map[string]workspaceMembership{}
	for _, ws := range frame.Value.Items {
		s.workspaceMemberships[ws.WorkspaceID] = ws
	}
	s.signalWorkspaceLocked()
	s.pinnedSessionIDs = append([]string(nil), frame.Value.PinnedSessionIds...)
	s.archivedSessionIDs = append([]string(nil), frame.Value.ArchivedSessionIds...)
	s.applyPinnedFlagsLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) applyWorkspacePinned(raw json.RawMessage) {
	var frame struct {
		PinnedSessionIds []string `json:"pinnedSessionIds"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.PinnedSessionIds == nil {
		return
	}
	s.setPinnedSessionIDs(frame.PinnedSessionIds)
}

func (s *Store) applyWorkspaceArchived(raw json.RawMessage) {
	var frame struct {
		ArchivedSessionIds []string `json:"archivedSessionIds"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.ArchivedSessionIds == nil {
		return
	}
	s.mu.Lock()
	s.archivedSessionIDs = append([]string(nil), frame.ArchivedSessionIds...)
	s.mu.Unlock()
	s.fire()
}

func itemFromListRow(it api.SessionListValueItemsItem) Item {
	row := Item{
		SessionID:      it.SessionId,
		ParentID:       derefString(it.ParentSessionId),
		Origin:         derefString(it.Origin),
		AgentAvailable: it.AgentAvailable,
		Title:          titleForItem(it),
		Running:        it.Running,
		Blank:          it.Blank,
		UpdatedAt:      it.UpdatedAt,
	}
	if it.Cwd != nil {
		row.Cwd = strings.TrimSpace(*it.Cwd)
	}
	return row
}
