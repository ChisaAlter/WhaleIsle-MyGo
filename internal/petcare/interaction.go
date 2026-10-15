package petcare

import (
	"math"
	"time"
)

func Poke(raw []byte, now time.Time) Stats {
	s := Snapshot(raw, now)
	s.Satiety = math.Round(s.Satiety)
	s.Mood = clamp(math.Round(s.Mood) + 2)
	s.Affection += careAffection(raw, "poke", 1, now)
	return s
}
func Throw(raw []byte, now time.Time) Stats {
	s := Snapshot(raw, now)
	s.Satiety = math.Round(s.Satiety)
	s.Mood = clamp(math.Round(s.Mood) - 9)
	s.Affection += careAffection(raw, "throw", 1, now)
	return s
}
