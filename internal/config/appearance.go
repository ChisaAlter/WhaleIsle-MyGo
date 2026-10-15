package config

import (
	"encoding/json"
	"whaleisle/internal/ui/theme"
)

const appearanceKey = "goAppearance"

func (s *Store) Appearance() (theme.Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := theme.Defaults()
	if raw := s.raw[appearanceKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return p, err
		}
	}
	return p.Normalized(), nil
}

// SaveAppearance commits native appearance preferences without changing unrelated config.
func (s *Store) SaveAppearance(p theme.Preferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}
	merged := map[string]json.RawMessage{}
	for key, value := range s.raw {
		merged[key] = value
	}
	merged[appearanceKey] = encoded
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
