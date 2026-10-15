package config

import "encoding/json"

// DesktopPet is the native sprite companion's opt-in configuration. It does
// not migrate or overwrite Electron's pet/live2dPet state.
type DesktopPet struct {
	Enabled               bool         `json:"enabled"`
	SkinID                string       `json:"skinId"`
	Neural                bool         `json:"neural,omitempty"`
	Behavior              PetBehavior  `json:"behavior"`
	PersonalityConfigured bool         `json:"personalityConfigured,omitempty"`
	Position              *PetPosition `json:"position,omitempty"`
}

func (s *Store) DesktopPet() DesktopPet {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := DesktopPet{Behavior: DefaultPetBehavior()}
	// Decode individually so a malformed field cannot enable the companion.
	var raw map[string]json.RawMessage
	if json.Unmarshal(s.raw["mygoDesktopPet"], &raw) == nil {
		_ = json.Unmarshal(raw["enabled"], &out.Enabled)
		_ = json.Unmarshal(raw["skinId"], &out.SkinID)
		_ = json.Unmarshal(raw["neural"], &out.Neural)
		_ = json.Unmarshal(raw["personalityConfigured"], &out.PersonalityConfigured)
		_ = json.Unmarshal(raw["behavior"], &out.Behavior)
		out.Position = decodePetPosition(raw["position"])
	}
	return out
}

func (s *Store) SaveDesktopPet(value DesktopPet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value.Behavior = NormalizePetBehavior(value.Behavior)
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

// SavedPetPersonality distinguishes a deliberately saved choice from fallback
// defaults materialized by another save (for example a drag-anchor migration).
// Malformed or missing fields never authorize overwriting the Host's persona.
func (s *Store) SavedPetPersonality() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var raw struct {
		Configured bool `json:"personalityConfigured"`
		Behavior   struct {
			Personality string `json:"personality"`
		} `json:"behavior"`
	}
	if json.Unmarshal(s.raw["mygoDesktopPet"], &raw) != nil || !raw.Configured {
		return "", false
	}
	switch raw.Behavior.Personality {
	case "natural", "genki", "tsundere", "poison":
		return raw.Behavior.Personality, true
	}
	return "", false
}
