package main

import (
	"fmt"
	"log"
	"sync"
	"time"
	"whaleisle/internal/ui/i18n"

	"whaleisle/internal/config"
)

// workspaceViewPreferences keeps the view's acknowledged snapshot separate
// from the configuration Store's current disk snapshot. Session view caches
// are retained after save, so independently merged fields are not new edits.
type workspaceViewPreferences struct {
	mu       sync.Mutex
	cond     *sync.Cond
	store    *config.Store
	observed map[string]config.WorkspaceNavigationPreferences

	// Single-writer debounced async persistence (enabled for live UI runs).
	async    bool
	debounce time.Duration
	onError  func(error)
	pending  map[string]config.WorkspaceNavigationPreferences
	inFlight map[string]config.WorkspaceNavigationPreferences
	timer    *time.Timer
	writing  bool
	lastErr  error
}

func newWorkspaceViewPreferences(store *config.Store) *workspaceViewPreferences {
	vp := &workspaceViewPreferences{
		store:    store,
		observed: map[string]config.WorkspaceNavigationPreferences{},
	}
	vp.cond = sync.NewCond(&vp.mu)
	return vp
}

func (s *workspaceViewPreferences) enableAsync(debounce time.Duration, onError func(error)) {
	if s == nil {
		return
	}
	if debounce <= 0 {
		debounce = asyncConfigDebounce
	}
	s.mu.Lock()
	s.async = true
	s.debounce = debounce
	s.onError = onError
	s.mu.Unlock()
}

func (s *workspaceViewPreferences) WorkspaceNavigationPreferences(host string) (config.WorkspaceNavigationPreferences, error) {
	s.mu.Lock()
	if s.async {
		if pending, ok := s.pending[host]; ok {
			out := cloneWorkspaceViewPreferences(pending)
			s.mu.Unlock()
			return out, nil
		}
		if inflight, ok := s.inFlight[host]; ok {
			out := cloneWorkspaceViewPreferences(inflight)
			s.mu.Unlock()
			return out, nil
		}
	}
	store := s.store
	s.mu.Unlock()

	value, err := store.ReadWorkspaceNavigationPreferences(host)
	s.mu.Lock()
	if err == nil {
		// Re-check pending/inFlight in case an async edit arrived during read.
		if s.async {
			if pending, ok := s.pending[host]; ok {
				out := cloneWorkspaceViewPreferences(pending)
				s.mu.Unlock()
				return out, nil
			}
			if inflight, ok := s.inFlight[host]; ok {
				out := cloneWorkspaceViewPreferences(inflight)
				s.mu.Unlock()
				return out, nil
			}
		}
		s.observed[host] = cloneWorkspaceViewPreferences(value)
	}
	s.mu.Unlock()
	return value, err
}

func (s *workspaceViewPreferences) SaveWorkspaceNavigationPreferences(host string, next config.WorkspaceNavigationPreferences) error {
	s.mu.Lock()
	base, ok := s.observed[host]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("%s", i18n.T(i18n.KeyCmdViewPreferencesBaselineMissing))
	}
	if s.async {
		if s.pending == nil {
			s.pending = map[string]config.WorkspaceNavigationPreferences{}
		}
		s.pending[host] = cloneWorkspaceViewPreferences(next)
		if s.timer != nil {
			s.timer.Reset(s.debounce)
		} else {
			s.timer = time.AfterFunc(s.debounce, func() {
				go s.drainAsync()
			})
		}
		s.mu.Unlock()
		return nil
	}
	defer s.mu.Unlock()
	if _, err := s.store.MergeWorkspaceNavigationPreferences(host, base, next); err != nil {
		return err
	}
	s.observed[host] = cloneWorkspaceViewPreferences(next)
	return nil
}

func (s *workspaceViewPreferences) drainAsync() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.writing {
		s.mu.Unlock()
		return
	}
	s.writing = true
	s.drainAsyncLocked()
	s.writing = false
	s.cond.Broadcast()
	s.mu.Unlock()
}

// drainAsyncLocked flushes queued host preference merges off the UI thread.
// REQUIRES: s.mu held and s.writing == true.
func (s *workspaceViewPreferences) drainAsyncLocked() {
	for len(s.pending) > 0 {
		if s.timer != nil {
			s.timer.Stop()
			s.timer = nil
		}
		batch := s.pending
		s.pending = nil
		s.inFlight = batch
		bases := make(map[string]config.WorkspaceNavigationPreferences, len(batch))
		for host := range batch {
			if base, ok := s.observed[host]; ok {
				bases[host] = cloneWorkspaceViewPreferences(base)
			}
		}
		store := s.store
		onErr := s.onError
		s.mu.Unlock()

		var batchErr error
		succeeded := make(map[string]config.WorkspaceNavigationPreferences, len(batch))
		for host, next := range batch {
			base, ok := bases[host]
			if !ok {
				continue
			}
			if _, err := store.MergeWorkspaceNavigationPreferences(host, base, next); err != nil {
				batchErr = err
				if onErr != nil {
					onErr(err)
				} else {
					log.Printf("workspace view preferences: async save %s: %v", host, err)
				}
				continue
			}
			succeeded[host] = next
		}

		s.mu.Lock()
		for host, saved := range succeeded {
			s.observed[host] = cloneWorkspaceViewPreferences(saved)
		}
		s.inFlight = nil
		s.lastErr = batchErr
	}
}

// Flush synchronously persists all pending async workspace view preference
// edits before shutdown or Host restart.
func (s *workspaceViewPreferences) Flush() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.async {
		return nil
	}
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	for s.writing {
		s.cond.Wait()
	}
	if len(s.pending) > 0 {
		s.writing = true
		s.drainAsyncLocked()
		s.writing = false
		s.cond.Broadcast()
	}
	return s.lastErr
}

func cloneWorkspaceViewPreferences(value config.WorkspaceNavigationPreferences) config.WorkspaceNavigationPreferences {
	copy := value
	copy.Collapsed = map[string]bool{}
	for key, collapsed := range value.Collapsed {
		copy.Collapsed[key] = collapsed
	}
	copy.SessionOrders = map[string][]string{}
	for key, order := range value.SessionOrders {
		copy.SessionOrders[key] = append([]string(nil), order...)
	}
	return copy
}
