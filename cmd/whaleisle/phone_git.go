package main

import (
	"path/filepath"
	"strings"
)

func (s *mainState) revokePhoneGit() {
	s.mu.Lock()
	s.phoneGitSession = ""
	s.phoneGitRoot = ""
	s.phoneGitWorkspaces = phoneGitWorkspaceScope{}
	r := s.phoneRuntime
	s.mu.Unlock()
	if r != nil {
		r.RevokeGit()
	}
}
func (s *mainState) setPhoneGit(enabled bool) {
	s.revokePhoneGit()
	if !enabled {
		return
	}
	s.mu.Lock()
	sessions := s.sessions
	s.mu.Unlock()
	if sessions == nil || s.cfgStore == nil || !s.cfgStore.Get().RemoteEnabled {
		return
	}
	snap := sessions.Snapshot()
	if snap.SelectedID == "" || strings.TrimSpace(snap.SelectedCwd) == "" {
		return
	}
	root, err := filepath.EvalSymlinks(snap.SelectedCwd)
	if err != nil {
		return
	}
	root, err = filepath.Abs(root)
	if err != nil || root == filepath.VolumeName(root)+string(filepath.Separator) {
		return
	}
	s.mu.Lock()
	s.phoneGitSession = snap.SelectedID
	s.phoneGitRoot = root
	s.mu.Unlock()
}
func (s *mainState) phoneGitGrant() (bool, string) {
	s.mu.Lock()
	id, root, sessions := s.phoneGitSession, s.phoneGitRoot, s.sessions
	s.mu.Unlock()
	if id == "" || root == "" || sessions == nil {
		return false, ""
	}
	snap := sessions.Snapshot()
	current, err := filepath.EvalSymlinks(snap.SelectedCwd)
	if err == nil {
		current, err = filepath.Abs(current)
	}
	if s.cfgStore == nil || !s.cfgStore.Get().RemoteEnabled || snap.SelectedID != id || err != nil || current != root {
		// Expiration of the default current-session grant does not remove
		// separately selected registered roots in the same Host run.
		s.mu.Lock()
		s.phoneGitSession, s.phoneGitRoot = "", ""
		r := s.phoneRuntime
		s.mu.Unlock()
		if r != nil {
			r.RevokeGit()
		}
		return false, ""
	}
	return true, root
}
func (s *mainState) authorizePhoneGit(root, action string) bool {
	allowed := false
	for _, expected := range s.phoneGitRoots() {
		if root == expected {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	switch action {
	case "gitStatus", "gitFetchForStatus", "gitReadPullRequest", "gitInit", "gitDiff", "gitCommit", "gitPush", "gitPull", "gitCreateChangeRequest", "gitPublishRepository", "gitStatusEntries", "gitBranchList", "gitSwitchBranch", "gitCreateBranch":
		return true
	}
	return false
}
