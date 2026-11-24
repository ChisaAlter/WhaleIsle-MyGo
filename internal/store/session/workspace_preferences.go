package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/storelocale"
)

type WorkspacePreferenceStore interface {
	WorkspaceNavigationPreferences(string) (config.WorkspaceNavigationPreferences, error)
	SaveWorkspaceNavigationPreferences(string, config.WorkspaceNavigationPreferences) error
}

type asyncWorkspacePreferenceStore interface {
	SaveWorkspaceNavigationPreferencesAsync(string, config.WorkspaceNavigationPreferences) error
}

type flushableWorkspacePreferenceStore interface {
	Flush() error
}

const defaultWorkspacePrefDebounce = 150 * time.Millisecond

type workspacePrefSaveJob struct {
	store WorkspacePreferenceStore
	scope string
	prefs config.WorkspaceNavigationPreferences
	seq   uint64
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

	writeMu       sync.Mutex
	asyncMode     bool
	debounce      time.Duration
	debounceTimer *time.Timer
	saveSeq       uint64
	pendingSave   *workspacePrefSaveJob
	debouncedJob  *workspacePrefSaveJob
	hasPending    atomic.Bool
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
	if s.workspacePreferences.debounceTimer != nil {
		s.workspacePreferences.debounceTimer.Stop()
		s.workspacePreferences.debounceTimer = nil
	}
	s.workspacePreferences.pendingSave = nil
	s.workspacePreferences.debouncedJob = nil
	s.workspacePreferences.hasPending.Store(false)
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
	for _, key := range []string{"::scratch", "::other", PinnedWorkspaceGroup} {
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
		b.errors[scope] = prefixedFaultText("workspacePreferences.read", storelocale.Pick("工作区视图偏好读取失败，本次保留当前状态：", "Workspace view preferences could not be read; view state kept for this run: "), err)
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
		if key == "::scratch" || key == "::other" || key == PinnedWorkspaceGroup {
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

// SetWorkspacePreferencesAsync toggles single-writer debounced background
// persistence for any WorkspacePreferenceStore (even ones that only implement
// synchronous SaveWorkspaceNavigationPreferences). Stores implementing
// SaveWorkspaceNavigationPreferencesAsync (such as *config.Store) always run
// non-blocking without holding Store.mu across disk I/O.
func (s *Store) SetWorkspacePreferencesAsync(enabled bool) {
	s.mu.Lock()
	s.workspacePreferences.asyncMode = enabled
	s.mu.Unlock()
}

// SetWorkspacePreferencesDebounce sets the coalescing window for debounced
// background preference saves. Zero restores the default (150ms).
func (s *Store) SetWorkspacePreferencesDebounce(d time.Duration) {
	s.mu.Lock()
	s.workspacePreferences.debounce = d
	s.mu.Unlock()
}

// FlushWorkspacePreferences synchronously drains any pending or debounced
// workspace preference writes and flushes the underlying preference store if it
// supports Flush(). Call before process exit or restart to guarantee durability.
func (s *Store) FlushWorkspacePreferences() error {
	s.drainWorkspacePreferenceSave()
	var err error
	if saveErr := s.flushDebouncedWorkspacePreferenceSave(); saveErr != nil {
		err = saveErr
	}
	s.mu.Lock()
	store := s.workspacePreferences.store
	s.mu.Unlock()
	if f, ok := store.(flushableWorkspacePreferenceStore); ok {
		if flushErr := f.Flush(); flushErr != nil && err == nil {
			err = flushErr
		}
	}
	return err
}

func (s *Store) saveWorkspacePreferencesLocked() {
	s.invalidateViewLocked()
	b := &s.workspacePreferences
	if b.store == nil || b.scope == "" || !b.dirty[b.scope] || b.readFailed[b.scope] {
		return
	}
	b.saveSeq++
	b.pendingSave = &workspacePrefSaveJob{
		store: b.store,
		scope: b.scope,
		prefs: cloneWorkspacePreferences(b.cache[b.scope]),
		seq:   b.saveSeq,
	}
	b.hasPending.Store(true)
}

func (s *Store) drainWorkspacePreferenceSave() {
	if !s.workspacePreferences.hasPending.Load() {
		return
	}
	s.mu.Lock()
	b := &s.workspacePreferences
	job := b.pendingSave
	b.pendingSave = nil
	asyncMode := b.asyncMode
	delay := b.debounce
	if delay <= 0 {
		delay = defaultWorkspacePrefDebounce
	}
	if job == nil {
		b.hasPending.Store(false)
		s.mu.Unlock()
		return
	}
	if asyncMode {
		b.debouncedJob = job
		if b.debounceTimer != nil {
			b.debounceTimer.Reset(delay)
		} else {
			b.debounceTimer = time.AfterFunc(delay, func() {
				_ = s.flushDebouncedWorkspacePreferenceSave()
			})
		}
		b.hasPending.Store(false)
		s.mu.Unlock()
		return
	}
	b.hasPending.Store(false)
	s.mu.Unlock()

	// Execute persistence outside s.mu so readers and UI frames never stall on
	// disk I/O. Stores supporting SaveWorkspaceNavigationPreferencesAsync return
	// immediately after optimistic in-memory commit + background debounce.
	err := s.executeWorkspacePreferenceSave(job, true)
	if err != nil {
		// Error already recorded in s.workspaceLayout.view.PreferenceError under lock.
		_ = err
	}
}

func (s *Store) flushDebouncedWorkspacePreferenceSave() error {
	s.mu.Lock()
	b := &s.workspacePreferences
	if b.debounceTimer != nil {
		b.debounceTimer.Stop()
		b.debounceTimer = nil
	}
	job := b.debouncedJob
	b.debouncedJob = nil
	s.mu.Unlock()
	if job == nil {
		return nil
	}
	err := s.executeWorkspacePreferenceSave(job, false)
	s.fire()
	return err
}

func (s *Store) executeWorkspacePreferenceSave(job *workspacePrefSaveJob, useAsyncMethod bool) error {
	b := &s.workspacePreferences
	b.writeMu.Lock()
	var err error
	if asyncStore, ok := job.store.(asyncWorkspacePreferenceStore); ok && useAsyncMethod {
		err = asyncStore.SaveWorkspaceNavigationPreferencesAsync(job.scope, job.prefs)
	} else {
		err = job.store.SaveWorkspaceNavigationPreferences(job.scope, job.prefs)
	}
	b.writeMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if b.store != job.store || b.scope != job.scope {
		return err
	}
	if err != nil {
		b.errors[job.scope] = prefixedFaultText("workspacePreferences.save", storelocale.Pick("本次保留当前视图状态，但保存失败：", "View state kept for this run; saving failed: "), err)
		s.workspaceLayout.view.PreferenceError = b.errors[job.scope]
		s.invalidateViewLocked()
		return err
	}
	if b.saveSeq == job.seq {
		b.dirty[job.scope] = false
		if e := b.edits[job.scope]; e != nil {
			e.groupBy, e.orderBy, e.archivedFilter = nil, nil, nil
			e.collapsed, e.sessionOrders = nil, nil
			e.replaceSessionOrders = false
		}
	}
	b.errors[job.scope] = ""
	s.workspaceLayout.view.PreferenceError = ""
	s.invalidateViewLocked()
	return nil
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
	s.drainWorkspacePreferenceSave()
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
