package config

import (
	"encoding/json"
	"fmt"
)

const pageShortcutsKey = "goPageShortcuts"

// PageShortcuts decodes only requested native binding IDs. Future IDs and
// unrelated settings stay opaque, including non-string future metadata.
func (s *Store) PageShortcuts(ids []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source := s.raw
	if s.shortcutReload != nil {
		var err error
		source, _, err = s.shortcutSnapshotLocked()
		if err != nil {
			return nil, err
		}
	}
	root := map[string]json.RawMessage{}
	if raw := source[pageShortcutsKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return nil, err
		}
	}
	out := map[string]string{}
	for _, id := range ids {
		if raw, ok := root[id]; ok {
			var value string
			if string(raw) == "null" {
				return nil, fmt.Errorf("shortcut %s must be a string", id)
			}
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("shortcut %s: %w", id, err)
			}
			out[id] = value
		}
	}
	return out, nil
}

// SavePageShortcuts changes only the explicitly supplied native binding IDs.
// Ordinary config Update also preserves this raw key. Publish the new active
// bindings only after this atomic replacement succeeds.
func (s *Store) SavePageShortcuts(patch map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	root := map[string]json.RawMessage{}
	if raw := s.raw[pageShortcutsKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return err
		}
	}
	if root == nil {
		root = map[string]json.RawMessage{}
	}
	for id, value := range patch {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		root[id] = encoded
	}
	encoded, err := json.Marshal(root)
	if err != nil {
		return err
	}
	merged := map[string]json.RawMessage{}
	for key, value := range s.raw {
		merged[key] = value
	}
	merged[pageShortcutsKey] = encoded
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	unlock, err := s.lockCommit()
	if err != nil {
		return err
	}
	defer unlock()
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
