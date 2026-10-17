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
	case "snapshot":
		s.applyWorkspaceSnapshot(raw)
	case "baseline":
		s.applyWorkspaceBaseline(raw)
	case "pinned":
		s.applyWorkspacePinned(raw)
	default:
		if s.applyWorkspaceSessionPatch(raw) {
			return
		}
		// Accept list-shaped or nested session payloads without crashing.
		if s.tryMergeListItems(raw) {
			return
		}
		var wrap struct {
			Session json.RawMessage `json:"session"`
			Item    json.RawMessage `json:"item"`
		}
		if json.Unmarshal(raw, &wrap) == nil {
			if len(wrap.Session) > 0 && s.applyWorkspaceSessionPatch(wrap.Session) {
				return
			}
			if len(wrap.Item) > 0 && s.applyWorkspaceSessionPatch(wrap.Item) {
				return
			}
		}
	}
}

func (s *Store) applyWorkspaceSnapshot(raw json.RawMessage) {
	var snap struct {
		Items    []api.SessionListValueItemsItem `json:"items"`
		Sessions []api.SessionListValueItemsItem `json:"sessions"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return
	}
	items := snap.Items
	if len(items) == 0 {
		items = snap.Sessions
	}
	if len(items) == 0 {
		return
	}
	s.mergeListItems(items)
}

func (s *Store) tryMergeListItems(raw json.RawMessage) bool {
	var list struct {
		Items []api.SessionListValueItemsItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil || len(list.Items) == 0 {
		return false
	}
	s.mergeListItems(list.Items)
	return true
}

func (s *Store) applyWorkspaceSessionPatch(raw json.RawMessage) bool {
	var patch struct {
		SessionId       string  `json:"sessionId"`
		Title           *string `json:"title"`
		Cwd             *string `json:"cwd"`
		Running         *bool   `json:"running"`
		UpdatedAt       *float64
		ParentSessionId *string `json:"parentSessionId"`
		Projections     *api.SessionListValueItemsItemProjections `json:"projections"`
	}
	if err := json.Unmarshal(raw, &patch); err != nil || patch.SessionId == "" {
		return false
	}
	s.mu.Lock()
	idx := -1
	for i := range s.items {
		if s.items[i].SessionID == patch.SessionId {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		return false
	}
	if patch.Title != nil && strings.TrimSpace(*patch.Title) != "" {
		s.items[idx].Title = strings.TrimSpace(*patch.Title)
	} else if patch.Projections != nil {
		if t := titleFromProjections(patch.Projections.Values); t != "" {
			s.items[idx].Title = t
		}
	}
	if patch.Cwd != nil {
		s.items[idx].Cwd = strings.TrimSpace(*patch.Cwd)
	}
	if patch.Running != nil {
		s.items[idx].Running = *patch.Running
	}
	if patch.UpdatedAt != nil {
		s.items[idx].UpdatedAt = *patch.UpdatedAt
	}
	if patch.ParentSessionId != nil {
		s.items[idx].ParentID = *patch.ParentSessionId
	}
	s.mu.Unlock()
	s.fire()
	return true
}

func (s *Store) mergeListItems(from []api.SessionListValueItemsItem) {
	items := make([]Item, 0, len(from))
	for _, it := range from {
		items = append(items, itemFromListRow(it))
	}
	s.mu.Lock()
	selected := s.selectedID
	s.items = items
	s.applyPinnedFlagsLocked()
	if selected == "" && len(items) > 0 {
		s.selectedID = items[0].SessionID
	}
	s.mu.Unlock()
	s.fire()
}

func (s *Store) applyWorkspaceBaseline(raw json.RawMessage) {
	var frame struct {
		Value struct {
			PinnedSessionIds []string `json:"pinnedSessionIds"`
		} `json:"value"`
	}
	if err := json.Unmarshal(raw, &frame); err != nil {
		return
	}
	if len(frame.Value.PinnedSessionIds) > 0 {
		s.setPinnedSessionIDs(frame.Value.PinnedSessionIds)
	}
}

func (s *Store) applyWorkspacePinned(raw json.RawMessage) {
	var frame struct {
		PinnedSessionIds []string `json:"pinnedSessionIds"`
	}
	if err := json.Unmarshal(raw, &frame); err != nil {
		return
	}
	s.setPinnedSessionIDs(frame.PinnedSessionIds)
}

func itemFromListRow(it api.SessionListValueItemsItem) Item {
	row := Item{
		SessionID: it.SessionId,
		ParentID:  derefString(it.ParentSessionId),
		Title:     titleForItem(it),
		Running:   it.Running,
		Blank:     it.Blank,
		UpdatedAt: it.UpdatedAt,
	}
	if it.Cwd != nil {
		row.Cwd = strings.TrimSpace(*it.Cwd)
	}
	return row
}
