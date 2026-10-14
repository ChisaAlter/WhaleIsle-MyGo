package config

import (
	"encoding/json"
	"fmt"
	"time"

	"whaleisle/internal/petcare"
)

func rawObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	if len(raw) == 0 || string(raw) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return nil, fmt.Errorf("pet care: stored value is not an object")
	}
	return out, nil
}

// PetStats projects only the shared Electron care state. Preferences, growth,
// enable flags and all unknown pet fields remain owned by their existing code.
func (s *Store) PetStats(now time.Time) petcare.Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	pet, _ := rawObject(s.raw["live2dPet"])
	return petcare.Snapshot(pet["stats"], now)
}

// FeedPetFile commits the actual renderer's care('fileEat') action in the same
// config.json. No filename/path or file content is accepted or persisted.
// On a stale-write or disk error neither raw state nor the care award advances.
func (s *Store) FeedPetFile(now time.Time) error {
	return s.applyPetCare("fileEat", petcare.FeedFile, now)
}

// PatPet implements only the actual original status-menu/head-touch action.
// The action is fixed by native code; no page-supplied arbitrary care kind.
func (s *Store) PatPet(now time.Time) error {
	return s.applyPetCare("pat", petcare.Pat, now)
}

// WakePet persists the original explicit wake mood cost, never a sleep reward.
func (s *Store) WakePet(now time.Time) error {
	return s.applyPetCare("wake", petcare.Wake, now)
}

func (s *Store) applyPetCare(kind string, effect func([]byte, time.Time) petcare.Stats, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pet, err := rawObject(s.raw["live2dPet"])
	if err != nil {
		return err
	}
	stats, err := rawObject(pet["stats"])
	if err != nil {
		return err
	}
	care, err := rawObject(stats["care"])
	if err != nil {
		return err
	}
	next := effect(pet["stats"], now)
	fields := map[string]any{"satiety": next.Satiety, "mood": next.Mood, "affection": next.Affection, "lastTick": next.LastTick}
	for key, value := range fields {
		stats[key], err = json.Marshal(value)
		if err != nil {
			return err
		}
	}
	care[kind], err = json.Marshal(now.UnixMilli())
	if err != nil {
		return err
	}
	stats["care"], err = json.Marshal(care)
	if err != nil {
		return err
	}
	pet["stats"], err = json.Marshal(stats)
	if err != nil {
		return err
	}
	return s.commitPetLocked(pet)
}

// Caller holds s.mu; all care/growth awards use the existing atomic config writer.
func (s *Store) commitPetLocked(pet map[string]json.RawMessage) error {
	var err error
	merged := make(map[string]json.RawMessage, len(s.raw)+1)
	for key, value := range s.raw {
		merged[key] = value
	}
	merged["live2dPet"], err = json.Marshal(pet)
	if err != nil {
		return err
	}
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
