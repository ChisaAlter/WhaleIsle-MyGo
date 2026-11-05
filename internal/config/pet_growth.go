package config

import (
	"bytes"
	"encoding/json"
	"time"
	"whaleisle/internal/petcare"
	"whaleisle/internal/petgrowth"
)

func clonePetGrowthState(g petgrowth.State) petgrowth.State {
	out := g
	if g.Baseline != nil {
		b := *g.Baseline
		out.Baseline = &b
	}
	if g.Today != nil {
		d := *g.Today
		out.Today = &d
	}
	if g.Food != nil {
		seen := make(map[string]int64, len(g.Food.Seen))
		for k, v := range g.Food.Seen {
			seen[k] = v
		}
		out.Food = &petgrowth.Food{
			Version:   g.Food.Version,
			Source:    g.Food.Source,
			InputOnly: g.Food.InputOnly,
			Available: g.Food.Available,
			Seen:      seen,
		}
	}
	return out
}

func petGrowthStateEqual(a, b petgrowth.State) bool {
	if a.AccountingVersion != b.AccountingVersion || a.Points != b.Points || a.TokensFed != b.TokensFed || a.TokensSeen != b.TokensSeen {
		return false
	}
	if (a.Baseline == nil) != (b.Baseline == nil) || (a.Baseline != nil && *a.Baseline != *b.Baseline) {
		return false
	}
	if (a.Today == nil) != (b.Today == nil) || (a.Today != nil && *a.Today != *b.Today) {
		return false
	}
	if (a.Food == nil) != (b.Food == nil) {
		return false
	}
	if a.Food != nil {
		if a.Food.Source != b.Food.Source || a.Food.InputOnly != b.Food.InputOnly || a.Food.Version != b.Food.Version || a.Food.Available != b.Food.Available || len(a.Food.Seen) != len(b.Food.Seen) {
			return false
		}
		for k, v := range a.Food.Seen {
			if bv, ok := b.Food.Seen[k]; !ok || bv != v {
				return false
			}
		}
	}
	return true
}

func sameRawSlice(a, b json.RawMessage) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	if &a[0] == &b[0] {
		return true
	}
	return bytes.Equal(a, b)
}

func (s *Store) ensurePetCacheLocked() error {
	raw := s.raw["live2dPet"]
	if s.petCacheValid && sameRawSlice(s.petRawCache, raw) {
		return nil
	}
	pet, err := rawObject(raw)
	if err != nil {
		return err
	}
	g, err := petgrowth.Parse(pet["growth"])
	if err != nil {
		return err
	}
	s.petRawCache = raw
	s.petStatsRawCache = append(json.RawMessage(nil), pet["stats"]...)
	s.petGrowthCache = clonePetGrowthState(g)
	s.petCacheValid = true
	s.petGrowthSettled = false
	return nil
}

func (s *Store) PetGrowth(now time.Time) (petgrowth.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePetCacheLocked(); err != nil {
		return petgrowth.Snapshot{}, err
	}
	return petgrowth.View(s.petGrowthCache, now), nil
}

// RefreshPetGrowth persists original hashed settlement high-water marks. It
// does not consume food, award care or points, or touch any API account balance.
func (s *Store) RefreshPetGrowth(samples map[string]int64, now time.Time) error {
	return s.refreshPetGrowthUsage(petgrowth.Usage{Samples: samples, Source: "sessions"}, now, false)
}
func (s *Store) RefreshPetGrowthUsage(usage petgrowth.Usage, now time.Time) error {
	return s.refreshPetGrowthUsage(usage, now, true)
}
func (s *Store) refreshPetGrowthUsage(usage petgrowth.Usage, now time.Time, migrate bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePetCacheLocked(); err != nil {
		return err
	}
	var next petgrowth.State
	var err error
	if migrate {
		next, err = petgrowth.RefreshUsage(clonePetGrowthState(s.petGrowthCache), usage, now)
	} else {
		next, err = petgrowth.Refresh(clonePetGrowthState(s.petGrowthCache), usage.Samples, now)
	}
	if err != nil {
		return err
	}
	if s.petGrowthSettled && petGrowthStateEqual(s.petGrowthCache, next) {
		return nil
	}
	pet, err := rawObject(s.raw["live2dPet"])
	if err != nil {
		return err
	}
	if err = mergeGrowth(pet, next); err != nil {
		return err
	}
	if err = s.commitPetLocked(pet); err != nil {
		return err
	}
	s.petRawCache = append(json.RawMessage(nil), s.raw["live2dPet"]...)
	s.petStatsRawCache = append(json.RawMessage(nil), pet["stats"]...)
	s.petGrowthCache = clonePetGrowthState(next)
	s.petCacheValid = true
	s.petGrowthSettled = true
	return nil
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
	s.petRawCache = append(json.RawMessage(nil), s.raw["live2dPet"]...)
	s.petStatsRawCache = append(json.RawMessage(nil), pet["stats"]...)
	s.petGrowthCache = clonePetGrowthState(next)
	s.petCacheValid = true
	s.petGrowthSettled = true
	return fed, nil
}

// Retain other-shell customization and future fields, including inside food;
// only these original ledger fields are owned by this module.
func mergeGrowth(pet map[string]json.RawMessage, next petgrowth.State) error {
	growth, err := rawObject(pet["growth"])
	if err != nil {
		return err
	}
	for k, v := range map[string]any{"accountingVersion": next.AccountingVersion, "points": next.Points, "tokensFed": next.TokensFed, "tokensSeen": next.TokensSeen, "baseline": next.Baseline, "today": next.Today} {
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
		for k, v := range map[string]any{"source": next.Food.Source, "inputOnly": next.Food.InputOnly, "version": 1, "available": next.Food.Available, "seen": next.Food.Seen} {
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
