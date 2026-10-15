package config

import (
	"encoding/json"
	"whaleisle/internal/desktop/wallpapers"
)

const backgroundKey = "goWallpaperBackground"

func (s *Store) WallpaperBackground() (wallpapers.Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := wallpapers.Defaults()
	if raw := s.raw[backgroundKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return p, err
		}
	}
	return p, nil
}

// SaveAppearance commits native appearance preferences without changing unrelated config.
func (s *Store) SaveWallpaperBackground(p wallpapers.Preferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkBaseline(); err != nil {
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
	merged[backgroundKey] = encoded
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
