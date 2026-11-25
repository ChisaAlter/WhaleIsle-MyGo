package config

import (
	"encoding/json"
	"errors"
	"strings"
)

const workspaceNavigationKey = "goWorkspaceNavigation"

type WorkspaceNavigationPreferences struct {
	SessionOrders  map[string][]string `json:"sessionOrders,omitempty"`
	GroupBy        string              `json:"groupBy"`
	OrderBy        string              `json:"orderBy"`
	ArchivedFilter string              `json:"archivedFilter"`
	Collapsed      map[string]bool     `json:"collapsed"`
}

func (s *Store) WorkspaceNavigationPreferences(hostKey string) (WorkspaceNavigationPreferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.workspaceNavigationPreferencesLocked(hostKey)
}

func (s *Store) workspaceNavigationPreferencesLocked(hostKey string) (WorkspaceNavigationPreferences, error) {
	var out WorkspaceNavigationPreferences
	root := map[string]json.RawMessage{}
	if raw := s.raw[workspaceNavigationKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return out, err
		}
	}
	if raw := root[hostKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return out, err
		}
	}
	return out, nil
}

// SaveWorkspaceNavigationPreferences follows ordinary config conflict detection,
// unknown-key preservation and atomic replacement. It changes only this Host.
func (s *Store) SaveWorkspaceNavigationPreferences(hostKey string, prefs WorkspaceNavigationPreferences) error {
	if strings.TrimSpace(hostKey) == "" {
		return errors.New("workspace preference Host is missing")
	}
	unlock, err := s.beginSyncWrite()
	if err != nil {
		return err
	}
	defer unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	merged, err := s.prepareWorkspacePreferencesRawLocked(hostKey, prefs)
	if err != nil {
		return err
	}
	baseline, err := s.commitRawLocked(s.baseline, merged)
	if err != nil {
		return err
	}
	s.raw = merged
	s.baseline = baseline
	return nil
}
