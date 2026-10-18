package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
)

type workspaceMembership WorkspaceRow

// IsNoDirectory requires the Host's actual scratch root and absence of membership.
// An orphaned or directory-less header must never masquerade as a scratch session.
func (v ViewState) IsNoDirectory(it Item) bool {
	return v.ScratchCwd != "" && filepathClean(it.Cwd) == filepathClean(v.ScratchCwd) && !v.WorkspaceMembers[it.SessionID]
}

func (s *Store) signalWorkspaceLocked() {
	if s.workspaceChanged != nil {
		close(s.workspaceChanged)
	}
	s.workspaceChanged = make(chan struct{})
}

func (s *Store) resetWorkspaceLocked() {
	s.resetWorkspaceLayoutHostLocked()
	s.invalidateWorkspaceNavigationLocked()
	s.workspaceID = ""
	s.workspaceOrder = nil
	s.workspaceFollowGeneration++
	s.scratchCwd = ""
	s.workspaceReady = false
	s.workspaceErr = nil
	s.workspaceMemberships = nil
	s.signalWorkspaceLocked()
}

func (s *Store) workspaceMembersLocked() map[string]bool {
	members := map[string]bool{}
	for _, ws := range s.workspaceMemberships {
		for _, id := range ws.SessionIDs {
			members[id] = true
		}
	}
	return members
}

func (s *Store) noDirectoryCandidateLocked(id, cwd string) bool {
	if s.workspaceMembersLocked()[id] {
		return false
	}
	for _, archived := range s.archivedSessionIDs {
		if archived == id {
			return false
		}
	}
	for _, it := range s.items {
		if it.SessionID == id {
			return it.Blank && !it.Running && !it.Pinned && it.ParentID == "" && it.Origin != "subagent" && filepathClean(it.Cwd) == filepathClean(cwd)
		}
	}
	return false
}

// CreateNoDirectorySession waits for Workspace authority and returns an identity
// without changing selection: its UI owner can preserve drafts and newer navigation.
// Calls are serialized so reuse checks cannot race another scratch creation.
func (s *Store) CreateNoDirectorySession(ctx context.Context, excludeIDs ...string) (string, error) {
	s.noDirectoryMu.Lock()
	defer s.noDirectoryMu.Unlock()
	s.mu.Lock()
	client := s.client
	generation := s.eventsGeneration
	s.mu.Unlock()
	if client == nil {
		return "", ErrNoClient
	}
	var cwd string
	for {
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation {
			s.mu.Unlock()
			return "", errors.New("Host connection changed")
		}
		if s.workspaceErr != nil {
			err := s.workspaceErr
			s.mu.Unlock()
			return "", err
		}
		if s.workspaceReady {
			cwd = strings.TrimSpace(s.scratchCwd)
			s.mu.Unlock()
			if cwd == "" {
				return "", errors.New("workspace baseline has no scratchCwd")
			}
			break
		}
		if s.workspaceChanged == nil {
			s.workspaceChanged = make(chan struct{})
		}
		changed := s.workspaceChanged
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-changed:
		}
	}
	s.mu.Lock()
	var ids []string
	excluded := map[string]bool{}
	for _, id := range excludeIDs {
		excluded[id] = true
	}
	for _, it := range s.items {
		if !excluded[it.SessionID] && s.noDirectoryCandidateLocked(it.SessionID, cwd) {
			ids = append(ids, it.SessionID)
		}
	}
	s.mu.Unlock()
	for _, id := range ids {
		var out api.SessionBlankReuseValue
		err := client.Call(ctx, "session/blankReuse", map[string]any{"request": map[string]any{"sessionId": id}}, &out)
		if err != nil {
			return "", fmt.Errorf("check blank scratch session: %w", err)
		}
		s.mu.Lock()
		valid := s.client == client && s.eventsGeneration == generation && s.noDirectoryCandidateLocked(id, cwd)
		s.mu.Unlock()
		if out.Reusable && valid {
			return id, nil
		}
	}
	s.mu.Lock()
	valid := s.client == client && s.eventsGeneration == generation && s.scratchCwd == cwd
	s.mu.Unlock()
	if !valid {
		return "", errors.New("Host workspace changed")
	}
	var out api.SessionCreateValue
	if err := client.Call(ctx, "session/create", map[string]any{"request": api.SessionCreateRequest{Cwd: &cwd}}, &out); err != nil {
		return "", err
	}
	if out.SessionId == "" {
		return "", errors.New("session/create returned no session")
	}
	s.mu.Lock()
	if s.client != client || s.eventsGeneration != generation {
		s.mu.Unlock()
		return "", errors.New("Host connection changed")
	}
	found := false
	for _, it := range s.items {
		if it.SessionID == out.SessionId {
			found = true
			break
		}
	}
	if !found {
		s.items = append(s.items, Item{SessionID: out.SessionId, Cwd: cwd, Blank: true})
	}
	s.mu.Unlock()
	s.fire()
	return out.SessionId, nil
}

// WorkspaceCwd is the configured workspace, independent of the selected scratch
// session. Sidebar workspace management must continue to describe this directory.
func (s *Store) WorkspaceCwd() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cwd
}
