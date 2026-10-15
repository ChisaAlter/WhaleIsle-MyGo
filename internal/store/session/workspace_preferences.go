package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

type WorkspacePreferenceStore interface {
	WorkspaceNavigationPreferences(string) (config.WorkspaceNavigationPreferences, error)
	SaveWorkspaceNavigationPreferences(string, config.WorkspaceNavigationPreferences) error
}
type workspacePreferenceBinding struct {
	store                WorkspacePreferenceStore
	authority, scope     string
	cache                map[string]config.WorkspaceNavigationPreferences
	dirty                map[string]bool
	readFailed           map[string]bool
	editedBeforeBaseline bool
}

func preferenceHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
func workspacePreferenceKey(row WorkspaceRow) string {
	return preferenceHash("workspace", row.WorkspaceID, row.Path)
}
func cloneWorkspacePreferences(in config.WorkspaceNavigationPreferences) config.WorkspaceNavigationPreferences {
	out := in
	out.Collapsed = map[string]bool{}
	for k, v := range in.Collapsed {
		out.Collapsed[k] = v
	}
	return out
}

// BindWorkspacePreferences binds the actual connected client, never credentials.
// authority may name a known supervised dsh home; remote callers can leave it
// empty to scope by the real endpoint origin. The Host scratch root further
// separates registry namespaces. Store implementations must not call back here.
func (s *Store) BindWorkspacePreferences(client *dsh.Client, authority string, store WorkspacePreferenceStore) {
	s.mu.Lock()
	if s.client != client || client == nil {
		s.mu.Unlock()
		return
	}
	if strings.TrimSpace(authority) == "" {
		authority = client.Endpoint.Origin
	}
	s.workspacePreferences.store = store
	s.workspacePreferences.authority = authority
	if s.workspacePreferences.cache == nil {
		s.workspacePreferences.cache = map[string]config.WorkspaceNavigationPreferences{}
		s.workspacePreferences.dirty = map[string]bool{}
		s.workspacePreferences.readFailed = map[string]bool{}
	}
	s.restoreWorkspacePreferencesLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) resetWorkspacePreferencesHostLocked() {
	s.workspacePreferences.scope = ""
	s.workspacePreferences.authority = ""
	s.workspacePreferences.store = nil
	s.workspacePreferences.editedBeforeBaseline = false
	s.workspaceLayout.view.GroupBy = ""
	s.workspaceLayout.view.OrderBy = ""
	s.workspaceLayout.view.PreferenceError = ""
}
func (s *Store) restoreWorkspacePreferencesLocked() {
	b := &s.workspacePreferences
	if !s.workspaceReady || b.store == nil || b.authority == "" {
		return
	}
	scope := preferenceHash("host", b.authority, s.scratchCwd)
	newScope := b.scope != scope
	if newScope {
		previousGroup, previousOrder := s.workspaceLayout.view.GroupBy, s.workspaceLayout.view.OrderBy
		previousCollapsed := s.workspaceLayout.view.Collapsed
		s.workspaceLayout.view.GroupBy = ""
		s.workspaceLayout.view.OrderBy = ""
		s.workspaceLayout.view.PreferenceError = ""
		prefs, cached := b.cache[scope]
		if !cached || (b.readFailed[scope] && !b.dirty[scope]) {
			loaded, err := b.store.WorkspaceNavigationPreferences(scope)
			b.readFailed[scope] = err != nil
			if err != nil {
				s.workspaceLayout.view.PreferenceError = "Workspace view preferences could not be read: " + err.Error()
			} else {
				prefs = loaded
			}
		}
		prefs = cloneWorkspacePreferences(prefs)
		if b.editedBeforeBaseline {
			if previousGroup != "" {
				prefs.GroupBy = previousGroup
			}
			if previousOrder != "" {
				prefs.OrderBy = previousOrder
			}
			for _, row := range s.workspaceRowsLocked() {
				if value, ok := previousCollapsed[row.WorkspaceID]; ok {
					prefs.Collapsed[workspacePreferenceKey(row)] = value
				}
			}
			for _, key := range []string{"::scratch", "::other"} {
				if value, ok := previousCollapsed[key]; ok {
					prefs.Collapsed[key] = value
				}
			}
			b.dirty[scope] = true
		}
		b.scope = scope
		b.cache[scope] = prefs
		b.editedBeforeBaseline = false
	}
	prefs := b.cache[scope]
	if prefs.GroupBy == "workspace" || prefs.GroupBy == "workspace-tree" || prefs.GroupBy == "flat" {
		s.workspaceLayout.view.GroupBy = prefs.GroupBy
	}
	if prefs.OrderBy == "manual" || prefs.OrderBy == "updated" {
		s.workspaceLayout.view.OrderBy = prefs.OrderBy
	}
	s.workspaceLayout.view.Collapsed = map[string]bool{}
	for _, row := range s.workspaceRowsLocked() {
		if value, ok := prefs.Collapsed[workspacePreferenceKey(row)]; ok {
			s.workspaceLayout.view.Collapsed[row.WorkspaceID] = value
		}
	}
	for _, key := range []string{"::scratch", "::other"} {
		if value, ok := prefs.Collapsed[key]; ok {
			s.workspaceLayout.view.Collapsed[key] = value
		}
	}
	if newScope && b.dirty[scope] {
		s.saveWorkspacePreferencesLocked()
	}
}
func (s *Store) workspacePreferencesChangedLocked() {
	b := &s.workspacePreferences
	if !s.workspaceReady || b.scope == "" || b.store == nil {
		b.editedBeforeBaseline = true
		return
	}
	prefs := cloneWorkspacePreferences(b.cache[b.scope])
	layout := s.workspaceLayoutLocked()
	prefs.GroupBy = layout.GroupBy
	prefs.OrderBy = layout.OrderBy
	for _, row := range s.workspaceRowsLocked() {
		if value, ok := layout.Collapsed[row.WorkspaceID]; ok {
			prefs.Collapsed[workspacePreferenceKey(row)] = value
		}
	}
	for _, key := range []string{"::scratch", "::other"} {
		if value, ok := layout.Collapsed[key]; ok {
			prefs.Collapsed[key] = value
		}
	}
	b.cache[b.scope] = prefs
	b.dirty[b.scope] = true
	s.saveWorkspacePreferencesLocked()
}
func (s *Store) saveWorkspacePreferencesLocked() {
	b := &s.workspacePreferences
	if b.store == nil || b.scope == "" || !b.dirty[b.scope] {
		return
	}
	if err := b.store.SaveWorkspaceNavigationPreferences(b.scope, cloneWorkspacePreferences(b.cache[b.scope])); err != nil {
		s.workspaceLayout.view.PreferenceError = "View state kept for this run; saving failed: " + err.Error()
		return
	}
	b.dirty[b.scope] = false
	b.readFailed[b.scope] = false
	s.workspaceLayout.view.PreferenceError = ""
}
func (s *Store) RetryWorkspacePreferences() {
	s.mu.Lock()
	b := &s.workspacePreferences
	if b.scope != "" && b.readFailed[b.scope] && !b.dirty[b.scope] {
		b.scope = ""
		s.restoreWorkspacePreferencesLocked()
	} else {
		s.saveWorkspacePreferencesLocked()
	}
	s.mu.Unlock()
	s.fire()
}

// InsertWorkspaceBeforeForHost is the drag boundary: an old drag may never
// mutate a newly attached Host that happens to reuse the same workspace ids.
func (s *Store) InsertWorkspaceBeforeForHost(ctx context.Context, generation uint64, id, before string) {
	s.reorderWorkspaceForHost(ctx, generation, id, before)
}

// WorkspaceDropAnchor places a row before/after a registered sibling. It rejects
// self drops and cross-parent moves without inventing a reparenting RPC.
func WorkspaceDropAnchor(groups []WorkspaceGroup, source, target string, after bool) (string, bool) {
	if source == target {
		return "", false
	}
	var sourceGroup, targetGroup *WorkspaceGroup
	for i := range groups {
		g := &groups[i]
		if g.Kind != "workspace" {
			continue
		}
		if g.Key == source {
			sourceGroup = g
		}
		if g.Key == target {
			targetGroup = g
		}
	}
	if sourceGroup == nil || targetGroup == nil || sourceGroup.ParentID != targetGroup.ParentID {
		return "", false
	}
	anchor := target
	if after {
		anchor = ""
		passed := false
		for _, g := range groups {
			if g.Kind != "workspace" || g.ParentID != targetGroup.ParentID {
				continue
			}
			if passed {
				anchor = g.Key
				break
			}
			if g.Key == target {
				passed = true
			}
		}
	}
	if anchor == source {
		return "", false
	}
	var siblings []string
	for _, g := range groups {
		if g.Kind == "workspace" && g.ParentID == sourceGroup.ParentID {
			siblings = append(siblings, g.Key)
		}
	}
	for i, id := range siblings {
		if id == source {
			next := ""
			if i+1 < len(siblings) {
				next = siblings[i+1]
			}
			if next == anchor {
				return "", false
			}
			break
		}
	}
	return anchor, true
}
