package main

import (
	"fmt"
	"sync"

	"whaleisle/internal/config"
)

// workspaceViewPreferences keeps the view's acknowledged snapshot separate
// from the configuration Store's current disk snapshot. Session view caches
// are retained after save, so independently merged fields are not new edits.
type workspaceViewPreferences struct {
	mu       sync.Mutex
	store    *config.Store
	observed map[string]config.WorkspaceNavigationPreferences
}

func newWorkspaceViewPreferences(store *config.Store) *workspaceViewPreferences {
	return &workspaceViewPreferences{store: store, observed: map[string]config.WorkspaceNavigationPreferences{}}
}

func (s *workspaceViewPreferences) WorkspaceNavigationPreferences(host string) (config.WorkspaceNavigationPreferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.store.ReadWorkspaceNavigationPreferences(host)
	if err == nil {
		s.observed[host] = cloneWorkspaceViewPreferences(value)
	}
	return value, err
}

func (s *workspaceViewPreferences) SaveWorkspaceNavigationPreferences(host string, next config.WorkspaceNavigationPreferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	base, ok := s.observed[host]
	if !ok {
		return fmt.Errorf("view preferences have no readable baseline for this Host")
	}
	if _, err := s.store.MergeWorkspaceNavigationPreferences(host, base, next); err != nil {
		return err
	}
	s.observed[host] = cloneWorkspaceViewPreferences(next)
	return nil
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
