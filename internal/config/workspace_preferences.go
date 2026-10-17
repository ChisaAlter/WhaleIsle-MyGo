package config

import (
	"encoding/json"
	"errors"
	"strings"
)

const workspaceNavigationKey = "goWorkspaceNavigation"

type WorkspaceNavigationPreferences struct {
	SessionOrders map[string][]string `json:"sessionOrders,omitempty"`
	GroupBy       string              `json:"groupBy"`
	OrderBy       string              `json:"orderBy"`
	Collapsed     map[string]bool     `json:"collapsed"`
}

func (s *Store) WorkspaceNavigationPreferences(hostKey string) (WorkspaceNavigationPreferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	root := map[string]json.RawMessage{}
	if raw := s.raw[workspaceNavigationKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return err
		}
	}
	if root == nil {
		root = map[string]json.RawMessage{}
	}
	host := map[string]json.RawMessage{}
	if raw := root[hostKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &host); err != nil {
			return err
		}
	}
	if host == nil {
		host = map[string]json.RawMessage{}
	}
	encoded, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	var known map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &known); err != nil {
		return err
	}
	for k, v := range known {
		host[k] = v
	}
	hostRaw, err := json.Marshal(host)
	if err != nil {
		return err
	}
	root[hostKey] = hostRaw
	rootRaw, err := json.Marshal(root)
	if err != nil {
		return err
	}
	merged := map[string]json.RawMessage{}
	for k, v := range s.raw {
		merged[k] = v
	}
	merged[workspaceNavigationKey] = rootRaw
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	commitMu.Lock()
	defer commitMu.Unlock()
	if err = s.checkBaseline(); err != nil {
		return err
	}
	if err = writeAtomic(s.path, data, s.rename); err != nil {
		return err
	}
	s.raw = merged
	s.baseline = fileBaseline{exists: true, data: data}
	return nil
}
