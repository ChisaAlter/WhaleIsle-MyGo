package config

import (
	"encoding/json"
	"time"
	"whaleisle/internal/petcare"
	"whaleisle/internal/petgrowth"
)

func (s *Store) PetGrowth(now time.Time) (petgrowth.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pet, err := rawObject(s.raw["live2dPet"])
	if err != nil {
		return petgrowth.Snapshot{}, err
	}
	g, err := petgrowth.Parse(pet["growth"])
	if err != nil {
		return petgrowth.Snapshot{}, err
	}
	return petgrowth.View(g, now), nil
}

// RefreshPetGrowth persists original hashed settlement high-water marks. It
// does not consume food, award care or points, or touch any API account balance.
func (s *Store) RefreshPetGrowth(samples map[string]int64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pet, err := rawObject(s.raw["live2dPet"])
	if err != nil {
		return err
	}
	g, err := petgrowth.Parse(pet["growth"])
	if err != nil {
		return err
	}
	next, err := petgrowth.Refresh(g, samples, now)
	if err != nil {
		return err
	}
	if err = mergeGrowth(pet, next); err != nil {
		return err
	}
	return s.commitPetLocked(pet)
}

// FeedPetTokens consumes at most the next tier's gap. Growth, remaining food,
// lifetime feeding and the real feedToken care effect commit together. A disk
// or stale-write failure leaves every value unchanged, including in memory.
func (s *Store) FeedPetTokens(now time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pet, err := rawObject(s.raw["live2dPet"])
	if err != nil {
		return 0, err
	}
	g, err := petgrowth.Parse(pet["growth"])
	if err != nil {
		return 0, err
	}
	next, fed, err := petgrowth.Feed(g, now)
	if err != nil || fed == 0 {
		return 0, err
	}
	stats, err := rawObject(pet["stats"])
	if err != nil {
		return 0, err
	}
	care, err := rawObject(stats["care"])
	if err != nil {
		return 0, err
	}
	effect := petcare.FeedToken(pet["stats"], now)
	for k, v := range map[string]any{"satiety": effect.Satiety, "mood": effect.Mood, "affection": effect.Affection, "lastTick": effect.LastTick} {
		stats[k], err = json.Marshal(v)
		if err != nil {
			return 0, err
		}
	}
	care["feedToken"], _ = json.Marshal(now.UnixMilli())
	stats["care"], err = json.Marshal(care)
	if err != nil {
		return 0, err
	}
	pet["stats"], err = json.Marshal(stats)
	if err != nil {
		return 0, err
	}
	if err = mergeGrowth(pet, next); err != nil {
		return 0, err
	}
	if err = s.commitPetLocked(pet); err != nil {
		return 0, err
	}
	return fed, nil
}

// Retain other-shell customization and future fields, including inside food;
// only these original ledger fields are owned by this module.
func mergeGrowth(pet map[string]json.RawMessage, next petgrowth.State) error {
	growth, err := rawObject(pet["growth"])
	if err != nil {
		return err
	}
	for k, v := range map[string]any{"points": next.Points, "tokensFed": next.TokensFed, "tokensSeen": next.TokensSeen, "baseline": next.Baseline, "today": next.Today} {
		growth[k], err = json.Marshal(v)
		if err != nil {
			return err
		}
	}
	if next.Food != nil {
		food, err := rawObject(growth["food"])
		if err != nil {
			return err
		}
		for k, v := range map[string]any{"version": 1, "available": next.Food.Available, "seen": next.Food.Seen} {
			food[k], err = json.Marshal(v)
			if err != nil {
				return err
			}
		}
		growth["food"], err = json.Marshal(food)
		if err != nil {
			return err
		}
	}
	pet["growth"], err = json.Marshal(growth)
	return err
}
