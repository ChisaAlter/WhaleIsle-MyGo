package remote

import (
	"fmt"
	"whaleisle/internal/dsh"
)

type fileCloseSave struct {
	key        string
	client     *dsh.Client
	generation uint64
	current    func() bool
	resources  []FileResource
	index      int
}

// preflightCloseSaveLocked captures the current draft metadata and validates
// every owner without selecting a resource, queuing a write or changing closeSave.
func (s *Files) preflightCloseSaveLocked() (*fileCloseSave, error) {
	s.syncDraftsLocked()
	v := s.states[s.key]
	if s.closeSave != nil {
		return nil, fmt.Errorf("Remote close save is already pending")
	}
	if v != nil {
		stashResource(v)
	}
	for key, state := range s.states {
		if key == s.key {
			continue
		}
		if state.Dirty() {
			return nil, fmt.Errorf("Return to session %s to save its retained remote draft before closing", state.SessionID)
		}
		for _, r := range state.Tabs {
			if r.Dirty() {
				return nil, fmt.Errorf("Return to session %s to save its retained remote draft before closing", state.SessionID)
			}
		}
	}
	if v == nil {
		return nil, nil
	}
	var resources []FileResource
	for _, r := range v.Tabs {
		if r.Dirty() {
			resources = append(resources, r)
		}
	}
	if len(resources) == 0 {
		return nil, nil
	}
	if s.client == nil || !v.Bound || v.Busy || v.Resolving || s.current != nil && !s.current() {
		return nil, fmt.Errorf("Remote session is not ready; drafts retained")
	}
	for _, r := range resources {
		if !r.Loaded || r.Stale || r.File.Binary || r.File.Truncated || r.File.Mtime == nil || len([]byte(r.Draft)) > filePreviewLimit {
			return nil, fmt.Errorf("Remote file %s needs reconciliation before closing; draft retained", r.Path)
		}
	}
	return &fileCloseSave{key: s.key, client: s.client, generation: s.generation, current: s.current, resources: resources}, nil
}

// PreflightDirtyForClose checks all retained drafts without scheduling writes.
// Mixed native panes call both local and remote preflights before saving either.
func (s *Files) PreflightDirtyForClose() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.preflightCloseSaveLocked()
	return err
}

// SaveDirtyForClose begins an explicit pane-close save. Cached drafts whose
// session is no longer selected cannot borrow the active session's authority.
func (s *Files) SaveDirtyForClose() error {
	s.mu.Lock()
	plan, err := s.preflightCloseSaveLocked()
	if err != nil || plan == nil {
		s.mu.Unlock()
		return err
	}
	s.closeSave = plan
	restoreResource(s.states[plan.key], plan.resources[0])
	s.mu.Unlock()
	s.Save()
	return nil
}

// CheckDirtyForClose advances one acknowledged write at a time. False/nil
// means pending. Any error ends this close attempt without discarding drafts.
func (s *Files) CheckDirtyForClose() (bool, error) {
	s.mu.Lock()
	s.syncDraftsLocked()
	p := s.closeSave
	if p == nil {
		for _, v := range s.states {
			if v.Dirty() {
				s.mu.Unlock()
				return false, fmt.Errorf("Remote drafts remain unsaved")
			}
			for _, r := range v.Tabs {
				if r.Dirty() {
					s.mu.Unlock()
					return false, fmt.Errorf("Remote drafts remain unsaved")
				}
			}
		}
		s.mu.Unlock()
		return true, nil
	}
	fail := func(message string) (bool, error) {
		s.closeSave = nil
		s.mu.Unlock()
		return false, fmt.Errorf("%s", message)
	}
	if p.key != s.key || p.client != s.client || p.generation != s.generation || p.current != nil && !p.current() {
		return fail("Remote session changed during close save; pane and drafts retained")
	}
	v := s.states[p.key]
	if v == nil || !v.Bound {
		return fail("Remote session is no longer ready; pane and drafts retained")
	}
	if v.Busy {
		s.mu.Unlock()
		return false, nil
	}
	stashResource(v)
	want := p.resources[p.index]
	i := resourceIndex(v, want.Path)
	if i < 0 {
		return fail("Remote resource changed during close save; pane retained")
	}
	got := v.Tabs[i]
	if got.Error != "" {
		return fail(got.Error)
	}
	if got.Stale || !got.Loaded || got.File.Mtime == nil || got.File.Content != want.Draft || got.Dirty() {
		return fail("Remote save was not acknowledged with the expected content and mtime; pane and drafts retained")
	}
	p.index++
	if p.index < len(p.resources) {
		next := p.resources[p.index]
		i = resourceIndex(v, next.Path)
		if i < 0 || v.Tabs[i].Draft != next.Draft || v.Tabs[i].Stale {
			return fail("Remote draft changed during close save; pane retained")
		}
		restoreResource(v, v.Tabs[i])
		s.mu.Unlock()
		s.Save()
		return false, nil
	}
	s.closeSave = nil
	for _, state := range s.states {
		if state.Dirty() {
			s.mu.Unlock()
			return false, fmt.Errorf("New remote drafts remain; pane retained")
		}
		for _, r := range state.Tabs {
			if r.Dirty() {
				s.mu.Unlock()
				return false, fmt.Errorf("New remote drafts remain; pane retained")
			}
		}
	}
	s.mu.Unlock()
	return true, nil
}
