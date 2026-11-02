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
type workspacePreferenceEdits struct {
	groupBy, orderBy, archivedFilter *string
	collapsed                        map[string]bool
	sessionOrders                    map[string][]string
	replaceSessionOrders             bool
	unresolvedCollapsed              map[string]bool
}

type workspacePreferenceBinding struct {
	store            WorkspacePreferenceStore
	authority, scope string
	cache            map[string]config.WorkspaceNavigationPreferences
	dirty            map[string]bool
	readFailed       map[string]bool
	errors           map[string]string
	edits            map[string]*workspacePreferenceEdits
	beforeBaseline   workspacePreferenceEdits
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
	out.SessionOrders = cloneSessionOrders(in.SessionOrders)
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
		s.workspacePreferences.errors = map[string]string{}
		s.workspacePreferences.edits = map[string]*workspacePreferenceEdits{}
	}
	s.restoreWorkspacePreferencesLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) resetWorkspacePreferencesHostLocked() {
	s.invalidateViewLocked()
	s.workspacePreferences.scope = ""
	s.workspaceLayout.view.SessionOrders = nil
	s.workspacePreferences.authority = ""
	s.workspacePreferences.store = nil
	s.workspacePreferences.beforeBaseline = workspacePreferenceEdits{}
	s.workspaceLayout.view.GroupBy = ""
	s.workspaceLayout.view.OrderBy = ""
	s.workspaceLayout.view.ArchivedFilter = ""
	s.workspaceLayout.view.PreferenceError = ""
}

// overlay applies only explicit edits, never the defaults materialized by the UI.
func (e *workspacePreferenceEdits) overlay(prefs config.WorkspaceNavigationPreferences) config.WorkspaceNavigationPreferences {
	prefs = cloneWorkspacePreferences(prefs)
	if e == nil {
		return prefs
	}
	if e.groupBy != nil {
		prefs.GroupBy = *e.groupBy
	}
	if e.orderBy != nil {
		prefs.OrderBy = *e.orderBy
	}
	if e.archivedFilter != nil {
		prefs.ArchivedFilter = *e.archivedFilter
	}
	for key, value := range e.collapsed {
		prefs.Collapsed[key] = value
	}
	if e.replaceSessionOrders {
		prefs.SessionOrders = map[string][]string{}
	}
	for key, value := range e.sessionOrders {
		prefs.SessionOrders[key] = append([]string(nil), value...)
	}
	return prefs
}

func (e *workspacePreferenceEdits) merge(other *workspacePreferenceEdits) {
	if other.groupBy != nil {
		value := *other.groupBy
		e.groupBy = &value
	}
	if other.orderBy != nil {
		value := *other.orderBy
		e.orderBy = &value
	}
	if other.archivedFilter != nil {
		value := *other.archivedFilter
		e.archivedFilter = &value
	}
	if e.collapsed == nil {
		e.collapsed = map[string]bool{}
	}
	for key, value := range other.collapsed {
		e.collapsed[key] = value
	}
	if other.replaceSessionOrders {
		e.sessionOrders = map[string][]string{}
		e.replaceSessionOrders = true
	}
	if e.sessionOrders == nil {
		e.sessionOrders = map[string][]string{}
	}
	for key, value := range other.sessionOrders {
		e.sessionOrders[key] = append([]string(nil), value...)
	}
	if e.unresolvedCollapsed == nil {
		e.unresolvedCollapsed = map[string]bool{}
	}
	for key, value := range other.unresolvedCollapsed {
		e.unresolvedCollapsed[key] = value
	}
}

func (e *workspacePreferenceEdits) hasResolved() bool {
	return e != nil && (e.groupBy != nil || e.orderBy != nil || e.archivedFilter != nil || len(e.collapsed) > 0 || len(e.sessionOrders) > 0 || e.replaceSessionOrders)
}

// A workspace's durable key includes its path. Before the Host baseline arrives,
// retain a toggle by ID until its actual row is known; never guess a path.
func (s *Store) resolveWorkspacePreferenceEditsLocked(e *workspacePreferenceEdits) {
	if e == nil || len(e.unresolvedCollapsed) == 0 {
		return
	}
	if e.collapsed == nil {
		e.collapsed = map[string]bool{}
	}
	for _, row := range s.workspaceRowsLocked() {
		if value, ok := e.unresolvedCollapsed[row.WorkspaceID]; ok {
			e.collapsed[workspacePreferenceKey(row)] = value
			delete(e.unresolvedCollapsed, row.WorkspaceID)
		}
	}
}

func (s *Store) applyWorkspacePreferenceViewLocked(scope string) {
	b := &s.workspacePreferences
	prefs := b.cache[scope]
	s.workspaceLayout.view.GroupBy = ""
	s.workspaceLayout.view.OrderBy = ""
	if prefs.GroupBy == "workspace" || prefs.GroupBy == "workspace-tree" || prefs.GroupBy == "flat" {
		s.workspaceLayout.view.GroupBy = prefs.GroupBy
	}
	if prefs.OrderBy == "manual" || prefs.OrderBy == "updated" {
		s.workspaceLayout.view.OrderBy = prefs.OrderBy
	}
	s.workspaceLayout.view.ArchivedFilter = "default"
	if prefs.ArchivedFilter == "show" || prefs.ArchivedFilter == "only" {
		s.workspaceLayout.view.ArchivedFilter = prefs.ArchivedFilter
	}
	s.workspaceLayout.view.SessionOrders = cloneSessionOrders(prefs.SessionOrders)
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
	if e := b.edits[scope]; e != nil {
		for key, value := range e.unresolvedCollapsed {
			s.workspaceLayout.view.Collapsed[key] = value
		}
	}
	s.workspaceLayout.view.PreferenceError = b.errors[scope]
}

func (s *Store) readWorkspacePreferenceBaselineLocked(scope string) bool {
	b := &s.workspacePreferences
	prefs, err := b.store.WorkspaceNavigationPreferences(scope)
	if err != nil {
		b.readFailed[scope] = true
		b.errors[scope] = "Workspace view preferences could not be read; view state kept for this run: " + err.Error()
		return false
	}
	b.readFailed[scope] = false
	b.errors[scope] = ""
	b.cache[scope] = b.edits[scope].overlay(prefs)
	return true
}

func (s *Store) restoreWorkspacePreferencesLocked() {
	s.invalidateViewLocked()
	b := &s.workspacePreferences
	if !s.workspaceReady || b.store == nil || b.authority == "" {
		return
	}
	scope := preferenceHash("host", b.authority, s.scratchCwd)
	newScope := b.scope != scope
	if newScope {
		b.scope = scope
		if b.edits[scope] == nil {
			b.edits[scope] = &workspacePreferenceEdits{}
		}
		b.edits[scope].merge(&b.beforeBaseline)
		b.beforeBaseline = workspacePreferenceEdits{}
	}
	e := b.edits[scope]
	wasDirty := b.dirty[scope]
	unresolved := 0
	if e != nil {
		unresolved = len(e.unresolvedCollapsed)
		s.resolveWorkspacePreferenceEditsLocked(e)
	}
	resolvedPending := e != nil && len(e.unresolvedCollapsed) < unresolved
	if _, cached := b.cache[scope]; !cached {
		b.cache[scope] = e.overlay(config.WorkspaceNavigationPreferences{})
		b.dirty[scope] = e.hasResolved()
		s.readWorkspacePreferenceBaselineLocked(scope)
	} else {
		b.cache[scope] = e.overlay(b.cache[scope])
		b.dirty[scope] = b.dirty[scope] || e.hasResolved()
		if newScope && b.readFailed[scope] {
			s.readWorkspacePreferenceBaselineLocked(scope)
		}
	}
	s.applyWorkspacePreferenceViewLocked(scope)
	// Follow updates do not repeatedly retry failed writes. Scope return and
	// explicit retry are opportunities to recover an earlier read failure.
	if (newScope || (resolvedPending && !wasDirty)) && b.dirty[scope] && !b.readFailed[scope] {
		s.saveWorkspacePreferencesLocked()
	}
}

// kind and key identify the actual user action; unrelated view state is not an edit.
func (s *Store) workspacePreferencesChangedLocked(kind, key string) {
	s.invalidateViewLocked()
	b := &s.workspacePreferences
	var e *workspacePreferenceEdits
	if !s.workspaceReady || b.scope == "" || b.store == nil {
		e = &b.beforeBaseline
	} else {
		e = b.edits[b.scope]
		if e == nil {
			e = &workspacePreferenceEdits{}
			b.edits[b.scope] = e
		}
	}
	switch kind {
	case "group":
		value := s.workspaceLayout.view.GroupBy
		e.groupBy = &value
	case "order":
		value := s.workspaceLayout.view.OrderBy
		e.orderBy = &value
	case "ordering":
		value := s.workspaceLayout.view.OrderBy
		e.orderBy = &value
		e.sessionOrders = cloneSessionOrders(s.workspaceLayout.view.SessionOrders)
		e.replaceSessionOrders = true
	case "archive":
		value := s.workspaceLayout.view.ArchivedFilter
		e.archivedFilter = &value
	case "collapsed":
		value := s.workspaceLayout.view.Collapsed[key]
		if key == "::scratch" || key == "::other" {
			if e.collapsed == nil {
				e.collapsed = map[string]bool{}
			}
			e.collapsed[key] = value
		} else {
			resolved := false
			for _, row := range s.workspaceRowsLocked() {
				if row.WorkspaceID == key {
					if e.collapsed == nil {
						e.collapsed = map[string]bool{}
					}
					e.collapsed[workspacePreferenceKey(row)] = value
					delete(e.unresolvedCollapsed, key)
					resolved = true
					break
				}
			}
			if !resolved {
				if e.unresolvedCollapsed == nil {
					e.unresolvedCollapsed = map[string]bool{}
				}
				e.unresolvedCollapsed[key] = value
			}
		}
	case "sessionOrder":
		if e.sessionOrders == nil {
			e.sessionOrders = map[string][]string{}
		}
		e.sessionOrders[key] = append([]string(nil), s.workspaceLayout.view.SessionOrders[key]...)
	}
	if e == &b.beforeBaseline {
		return
	}
	b.cache[b.scope] = e.overlay(b.cache[b.scope])
	b.dirty[b.scope] = e.hasResolved()
	if b.dirty[b.scope] && !b.readFailed[b.scope] {
		s.saveWorkspacePreferencesLocked()
	}
}

func (s *Store) saveWorkspacePreferencesLocked() {
	s.invalidateViewLocked()
	b := &s.workspacePreferences
	if b.store == nil || b.scope == "" || !b.dirty[b.scope] || b.readFailed[b.scope] {
		return
	}
	if err := b.store.SaveWorkspaceNavigationPreferences(b.scope, cloneWorkspacePreferences(b.cache[b.scope])); err != nil {
		b.errors[b.scope] = "View state kept for this run; saving failed: " + err.Error()
		s.workspaceLayout.view.PreferenceError = b.errors[b.scope]
		return
	}
	b.dirty[b.scope] = false
	if e := b.edits[b.scope]; e != nil {
		e.groupBy, e.orderBy, e.archivedFilter = nil, nil, nil
		e.collapsed, e.sessionOrders = nil, nil
		e.replaceSessionOrders = false
	}
	b.errors[b.scope] = ""
	s.workspaceLayout.view.PreferenceError = ""
}

func (s *Store) RetryWorkspacePreferences() {
	s.mu.Lock()
	b := &s.workspacePreferences
	if b.scope != "" && b.store != nil {
		if !b.readFailed[b.scope] || s.readWorkspacePreferenceBaselineLocked(b.scope) {
			s.saveWorkspacePreferencesLocked()
		}
		s.applyWorkspacePreferenceViewLocked(b.scope)
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
