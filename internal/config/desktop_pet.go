package config

import "encoding/json"

// DesktopPet is the native sprite companion's opt-in configuration. It does
// not migrate or overwrite Electron's pet/live2dPet state.
type DesktopPet struct {
	Enabled bool   `json:"enabled"`
	SkinID  string `json:"skinId"`
	Neural  bool   `json:"neural,omitempty"`
}

func (s *Store) DesktopPet() DesktopPet {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out DesktopPet
	// Decode individually so a malformed field cannot enable the companion.
	var raw map[string]json.RawMessage
	if json.Unmarshal(s.raw["mygoDesktopPet"], &raw) == nil {
		_ = json.Unmarshal(raw["enabled"], &out.Enabled)
		_ = json.Unmarshal(raw["skinId"], &out.SkinID)
		_ = json.Unmarshal(raw["neural"], &out.Neural)
	}
	return out
}

func (s *Store) SaveDesktopPet(value DesktopPet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	merged := make(map[string]json.RawMessage, len(s.raw)+1)
	for k, v := range s.raw {
		merged[k] = v
	}
	merged["mygoDesktopPet"] = data
	data, err = json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	commitMu.Lock()
	defer commitMu.Unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	if err := writeAtomic(s.path, data, s.rename); err != nil {
		return err
	}
	s.raw = merged
	s.baseline = fileBaseline{exists: true, data: data}
	return nil
}
