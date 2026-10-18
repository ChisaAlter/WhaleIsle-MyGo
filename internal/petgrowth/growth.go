// Package petgrowth implements Electron's local token-food ledger. It has no
// account, billing, payment, model or network API. One already-used token earns
// one local food point; only an explicit meal consumes it for pet growth.
package petgrowth

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"time"
)

const MaxCount int64 = 9007199254740991 // interoperable with Electron's JS integers
var hashKey = regexp.MustCompile(`^[0-9a-f]{64}$`)
var Levels = []struct {
	At   int64
	Name string
}{
	{0, "幼鲸"}, {100000000, "小鲸"}, {300000000, "干饭鲸"}, {600000000, "鲸鱼娘"},
	{1200000000, "大肥鱼"}, {2200000000, "干饭大王"}, {4000000000, "米饭女帝"},
	{7000000000, "鲸吞四海"}, {12000000000, "星海饭皇"}, {20000000000, "干饭真神"},
}

type Day struct {
	Day  string `json:"day"`
	Used int64  `json:"used"`
}
type Food struct {
	Version   int              `json:"version"`
	Available int64            `json:"available"`
	Seen      map[string]int64 `json:"seen"`
}
type State struct {
	Points     int64  `json:"points"`
	TokensFed  int64  `json:"tokensFed"`
	TokensSeen int64  `json:"tokensSeen"`
	Baseline   *int64 `json:"baseline"`
	Today      *Day   `json:"today"`
	Food       *Food  `json:"food,omitempty"`
}
type Snapshot struct {
	Points, TokensFed, TokensSeen, Available, NextFeed, TodayUsed int64
	Level                                                         int
	LevelName                                                     string
	NextAt                                                        *int64
}

func count(raw json.RawMessage) (int64, bool, error) {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return 0, false, nil
	}
	n, ok := v.(float64)
	if !ok || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
		return 0, false, nil
	}
	if n > float64(MaxCount) {
		return 0, false, errors.New("pet growth count exceeds safe integer range")
	}
	return int64(math.Floor(n)), true, nil
}

// Parse normalizes the original schema; a malformed ledger must not silently
// become a fresh ledger and re-credit settlements. Other absent/invalid scalar
// fields retain Electron's zero/null defaults.
func Parse(raw []byte) (State, error) {
	var g State
	if len(raw) == 0 || string(raw) == "null" {
		return g, nil
	}
	var src map[string]json.RawMessage
	if err := json.Unmarshal(raw, &src); err != nil || src == nil {
		return g, errors.New("invalid pet growth state")
	}
	for k, dst := range map[string]*int64{"points": &g.Points, "tokensFed": &g.TokensFed, "tokensSeen": &g.TokensSeen} {
		v, _, err := count(src[k])
		if err != nil {
			return g, err
		}
		*dst = v
	}
	baseline, valid, err := count(src["baseline"])
	if err != nil {
		return g, err
	}
	if valid {
		g.Baseline = &baseline
	}
	var day map[string]json.RawMessage
	if json.Unmarshal(src["today"], &day) == nil {
		var key string
		_ = json.Unmarshal(day["day"], &key)
		if key != "" {
			n, _, err := count(day["used"])
			if err != nil {
				return g, err
			}
			g.Today = &Day{key, n}
		}
	}
	if food := src["food"]; len(food) > 0 && string(food) != "null" {
		var obj map[string]json.RawMessage
		var version any
		_ = json.Unmarshal(food, &obj)
		_ = json.Unmarshal(obj["version"], &version)
		if version != float64(1) {
			return g, errors.New("unsupported pet food ledger")
		}
		n, valid, err := count(obj["available"])
		if err != nil || !valid {
			return g, errors.New("invalid pet food balance")
		}
		var seen map[string]json.RawMessage
		if json.Unmarshal(obj["seen"], &seen) != nil || seen == nil {
			return g, errors.New("invalid pet food settlements")
		}
		g.Food = &Food{1, n, make(map[string]int64, len(seen))}
		for k, raw := range seen {
			n, valid, err := count(raw)
			if !hashKey.MatchString(k) || !valid || err != nil {
				return g, errors.New("invalid pet food settlement")
			}
			g.Food.Seen[k] = n
		}
	}
	return g, nil
}
func add(a, b int64) (int64, error) {
	if b < 0 || a < 0 || a > MaxCount-b {
		return 0, errors.New("pet growth total exceeds safe integer range")
	}
	return a + b, nil
}

// ValidateSamples accepts only numeric usage and hashed settlement identities,
// never messages, paths, session names or account balances.
func ValidateSamples(samples map[string]int64) (int64, error) {
	if samples == nil {
		return 0, errors.New("missing settlement samples")
	}
	var total int64
	for k, n := range samples {
		if !hashKey.MatchString(k) || n < 0 {
			return 0, errors.New("invalid settlement sample")
		}
		var err error
		total, err = add(total, n)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}
func Available(g State) int64 {
	if g.Food != nil {
		return g.Food.Available
	}
	if g.Baseline == nil {
		return 0
	}
	return max(0, g.TokensSeen-*g.Baseline-g.TokensFed)
}
func View(g State, now time.Time) Snapshot {
	level := 0
	for i, v := range Levels {
		if g.Points >= v.At {
			level = i
		}
	}
	v := Snapshot{Points: g.Points, TokensFed: g.TokensFed, TokensSeen: g.TokensSeen, Available: Available(g), Level: level + 1, LevelName: Levels[level].Name}
	v.NextFeed = v.Available
	if level+1 < len(Levels) {
		at := Levels[level+1].At
		v.NextAt = &at
		v.NextFeed = min(v.NextFeed, at-g.Points)
	}
	if g.Today != nil && g.Today.Day == now.Format("2006-01-02") {
		v.TodayUsed = g.Today.Used
	}
	return v
}

// Refresh keeps Electron's settlement keys and high-water marks. Missing or
// reduced samples never revoke food; returning samples never earn twice.
func Refresh(g State, samples map[string]int64, now time.Time) (State, error) {
	total, err := ValidateSamples(samples)
	if err != nil {
		return g, err
	}
	first := g.Baseline == nil
	baseline := max(int64(0), total-g.TokensFed)
	if !first {
		baseline = min(*g.Baseline, total)
	}
	delta := max(int64(0), total-g.TokensSeen)
	food := &Food{Version: 1, Seen: map[string]int64{}}
	if g.Food == nil {
		food.Available = max(int64(0), total-baseline-g.TokensFed)
		for k, n := range samples {
			food.Seen[k] = n
		}
	} else {
		food.Available = g.Food.Available
		for k, n := range g.Food.Seen {
			food.Seen[k] = n
		}
		delta = 0
		for k, n := range samples {
			var err error
			delta, err = add(delta, max(int64(0), n-food.Seen[k]))
			if err != nil {
				return g, err
			}
			food.Seen[k] = max(food.Seen[k], n)
		}
		food.Available, err = add(food.Available, delta)
		if err != nil {
			return g, err
		}
	}
	day := now.Format("2006-01-02")
	used := int64(0)
	if !first {
		if g.Today != nil && g.Today.Day == day {
			used = g.Today.Used
		}
		used, err = add(used, delta)
		if err != nil {
			return g, err
		}
	}
	g.Baseline = &baseline
	g.TokensSeen = total
	g.Food = food
	g.Today = &Day{day, used}
	return g, nil
}
func Feed(g State, now time.Time) (State, int64, error) {
	fed := View(g, now).NextFeed
	points, err := add(g.Points, fed)
	if err != nil {
		return g, 0, err
	}
	tokens, err := add(g.TokensFed, fed)
	if err != nil {
		return g, 0, err
	}
	g.Points = points
	g.TokensFed = tokens
	if g.Food != nil {
		food := *g.Food
		food.Available -= fed
		g.Food = &food
	}
	return g, fed, nil
}
