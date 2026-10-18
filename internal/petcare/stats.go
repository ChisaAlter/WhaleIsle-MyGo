// Package petcare implements the real fileEat/feedToken/pat care actions from
// Electron's pet-stats.js. It deliberately has no token ledger, file IO or renderer logic.
package petcare

import (
	"encoding/json"
	"math"
	"time"
)

type Stats struct {
	Satiety   float64 `json:"satiety"`
	Mood      float64 `json:"mood"`
	Affection float64 `json:"affection"`
	LastTick  float64 `json:"lastTick"`
}

func clamp(n float64) float64 { return math.Max(0, math.Min(100, n)) }

// Snapshot lazily folds elapsed time, including time while the app was closed.
// Original defaults are 70/70/0; affection never decays. Reads do not write.
func Snapshot(raw []byte, now time.Time) Stats {
	var values map[string]json.RawMessage
	_ = json.Unmarshal(raw, &values)
	number := func(key string, fallback float64) float64 {
		var value any
		if json.Unmarshal(values[key], &value) != nil {
			return fallback
		}
		n, ok := value.(float64)
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return fallback
		}
		return n
	}
	ms := float64(now.UnixMilli())
	s := Stats{Satiety: clamp(math.Round(number("satiety", 70))), Mood: clamp(math.Round(number("mood", 70))), Affection: math.Floor(math.Max(0, number("affection", 0))), LastTick: number("lastTick", ms)}
	if s.LastTick <= 0 {
		s.LastTick = ms
	}
	hours := math.Min(72, math.Max(0, (ms-s.LastTick)/3600000))
	if hours > 0 {
		s.Satiety = clamp(s.Satiety - 3*hours)
		distance := 62 - s.Mood
		s.Mood += math.Copysign(math.Min(math.Abs(distance), 8*hours), distance)
		s.LastTick = ms
	}
	return s
}

// FeedFile mirrors readStats → applyCare('fileEat') → writeStats. The original
// active drop path grants once per action (+2 satiety/+1 mood), not per file;
// it has no daily cap and no affection reward. Persistence normalizes to ints.
func FeedFile(raw []byte, now time.Time) Stats {
	s := Snapshot(raw, now)
	s.Satiety = clamp(math.Round(s.Satiety) + 2)
	s.Mood = clamp(math.Round(s.Mood) + 1)
	return s
}

// FeedToken is granted only after a nonzero local-food meal. Electron's same
// action cooldown reduces +5 affection to floor(5/4)=1 for ten minutes.
func FeedToken(raw []byte, now time.Time) Stats {
	s := Snapshot(raw, now)
	s.Satiety = clamp(math.Round(s.Satiety) + 45)
	s.Mood = clamp(math.Round(s.Mood) + 12)
	s.Affection += careAffection(raw, "feedToken", 5, now)
	return s
}

// Pat is the wired status-panel/head-touch action: +10 mood and +4 bond,
// with only +1 bond when pat was already used within the last ten minutes.
// It consumes no food, changes no growth points and never calls a model.
func Pat(raw []byte, now time.Time) Stats {
	s := Snapshot(raw, now)
	s.Satiety = math.Round(s.Satiety)
	s.Mood = clamp(math.Round(s.Mood) + 10)
	s.Affection += careAffection(raw, "pat", 4, now)
	return s
}

// The original cooldown is keyed by care kind, never by the most recent
// interaction of any kind. Reuse it for token feeding and explicit patting.
func careAffection(raw []byte, kind string, amount float64, now time.Time) float64 {
	var saved struct {
		Care map[string]json.RawMessage `json:"care"`
	}
	if json.Unmarshal(raw, &saved) == nil {
		var last any
		if json.Unmarshal(saved.Care[kind], &last) == nil {
			if n, ok := last.(float64); ok && n >= 0 && float64(now.UnixMilli())-n < 600000 {
				return math.Floor(amount * 0.25)
			}
		}
	}
	return amount
}

// Wake is the original explicit nap-menu wake action. Sleeping itself has no
// saved reward; cursor/feeding/patting wake paths do not apply this penalty.
func Wake(raw []byte, now time.Time) Stats {
	s := Snapshot(raw, now)
	s.Satiety = math.Round(s.Satiety)
	s.Mood = clamp(math.Round(s.Mood) - 7)
	return s
}
