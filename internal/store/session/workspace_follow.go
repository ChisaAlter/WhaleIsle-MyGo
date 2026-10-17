package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/sessioncost"
)

func (s *Store) FollowWorkspace(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	if mux == nil || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if s.workspaceFollowCancel != nil {
		s.workspaceFollowCancel()
	}
	s.workspaceFollowGeneration++
	generation, hostGeneration := s.workspaceFollowGeneration, s.eventsGeneration
	fctx, cancel := context.WithCancel(ctx)
	s.workspaceFollowCancel = cancel
	s.mu.Unlock()
	go s.runWorkspaceFollow(fctx, mux, generation, hostGeneration)
}
func (s *Store) workspaceFollowCurrentLocked(ctx context.Context, mux *dsh.Mux, generation, hostGeneration uint64) bool {
	return ctx.Err() == nil && s.mux == mux && s.workspaceFollowGeneration == generation && s.eventsGeneration == hostGeneration
}
func (s *Store) runWorkspaceFollow(ctx context.Context, mux *dsh.Mux, generation, hostGeneration uint64) {
	report := func(err error) {
		s.mu.Lock()
		if !s.workspaceFollowCurrentLocked(ctx, mux, generation, hostGeneration) {
			s.mu.Unlock()
			return
		}
		s.workspaceErr = err
		s.lastAction = "workspace follow: " + err.Error()
		s.signalWorkspaceLocked()
		s.mu.Unlock()
		s.fire()
	}
	if err := mux.Dial(ctx); err != nil {
		report(err)
		return
	}
	stream, err := mux.Open(ctx, "workspace/follow", map[string]any{"args": map[string]any{}})
	if err != nil {
		report(err)
		return
	}
	defer stream.Cancel()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				report(errors.New("workspace follow unavailable; reconnect and try again"))
			}
			return
		}
		s.mu.Lock()
		if !s.workspaceFollowCurrentLocked(ctx, mux, generation, hostGeneration) {
			s.mu.Unlock()
			return
		}
		changed := s.applyWorkspaceFrameLocked(raw)
		s.mu.Unlock()
		if changed {
			s.fire()
		}
	}
}
func (s *Store) applyWorkspaceFollowFrame(raw json.RawMessage) {
	s.mu.Lock()
	changed := s.applyWorkspaceFrameLocked(raw)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}
func (s *Store) applyWorkspaceBaseline(raw json.RawMessage) { s.applyWorkspaceFollowFrame(raw) }
func (s *Store) applyWorkspacePinned(raw json.RawMessage)   { s.applyWorkspaceFollowFrame(raw) }
func (s *Store) applyWorkspaceArchived(raw json.RawMessage) { s.applyWorkspaceFollowFrame(raw) }
func (s *Store) applyWorkspaceFrameLocked(raw json.RawMessage) bool {
	var frame struct {
		Type  string `json:"type"`
		Value *struct {
			ScratchCwd         string                `json:"scratchCwd"`
			Items              []workspaceMembership `json:"items"`
			PinnedSessionIDs   []string              `json:"pinnedSessionIds"`
			ArchivedSessionIDs []string              `json:"archivedSessionIds"`
		} `json:"value"`
		Workspace          workspaceMembership `json:"workspace"`
		WorkspaceID        string              `json:"workspaceId"`
		WorkspaceIDs       []string            `json:"workspaceIds"`
		PinnedSessionIDs   []string            `json:"pinnedSessionIds"`
		ArchivedSessionIDs []string            `json:"archivedSessionIds"`
	}
	if json.Unmarshal(raw, &frame) != nil {
		return false
	}
	switch frame.Type {
	case "baseline":
		if frame.Value == nil || frame.Value.PinnedSessionIDs == nil || frame.Value.ArchivedSessionIDs == nil {
			return false
		}
		s.scratchCwd = strings.TrimSpace(frame.Value.ScratchCwd)
		s.workspaceReady = true
		s.workspaceErr = nil
		s.workspaceMemberships = map[string]workspaceMembership{}
		s.workspaceOrder = nil
		for _, ws := range frame.Value.Items {
			if ws.WorkspaceID != "" {
				if _, exists := s.workspaceMemberships[ws.WorkspaceID]; !exists {
					s.workspaceOrder = append(s.workspaceOrder, ws.WorkspaceID)
				}
				s.workspaceMemberships[ws.WorkspaceID] = ws
			}
		}
		s.pinnedSessionIDs = append([]string(nil), frame.Value.PinnedSessionIDs...)
		s.archivedSessionIDs = append([]string(nil), frame.Value.ArchivedSessionIDs...)
		s.applyPinnedFlagsLocked()
	case "upsert":
		if frame.Workspace.WorkspaceID == "" {
			return false
		}
		s.putWorkspaceRowLocked(WorkspaceRow(frame.Workspace))
	case "remove":
		delete(s.workspaceMemberships, frame.WorkspaceID)
		s.removeWorkspaceOrderLocked(frame.WorkspaceID)
		if s.workspaceID == frame.WorkspaceID {
			s.workspaceID = ""
		}
	case "order":
		if frame.WorkspaceIDs == nil {
			return false
		}
		s.workspaceOrder = append([]string(nil), frame.WorkspaceIDs...)
	case "pinned":
		if frame.PinnedSessionIDs == nil {
			return false
		}
		s.pinnedSessionIDs = append([]string(nil), frame.PinnedSessionIDs...)
		s.applyPinnedFlagsLocked()
	case "archived":
		if frame.ArchivedSessionIDs == nil {
			return false
		}
		s.archivedSessionIDs = append([]string(nil), frame.ArchivedSessionIDs...)
	default:
		return false
	}
	s.restoreWorkspacePreferencesLocked()
	s.signalWorkspaceLocked()
	return true
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
	if it.Projections != nil {
		row.AgentMetrics = decodeAgentMetrics(it.Projections.Values, it.Projections.AsOfSeq)
		row.BilledUsage = sessioncost.Decode(it.Projections.Values)
		row.BilledUsageSeq = it.Projections.AsOfSeq
		row.BilledUsageKind = it.Projections.Kind
	}
	if it.Cwd != nil {
		row.Cwd = strings.TrimSpace(*it.Cwd)
	}
	return row
}
