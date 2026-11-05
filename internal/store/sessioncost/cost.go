// Package sessioncost reads Host cumulative billing projections. It never folds
// visible messages: paging, compaction and branch history cannot create usage.
package sessioncost

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"time"
	"whaleisle/internal/store/usage"
)

type Bucket struct {
	Miss   float64 `json:"missInputTokens"`
	Hit    float64 `json:"cacheReadTokens"`
	Output float64 `json:"outputTokens"`
}

func (b *Bucket) UnmarshalJSON(raw []byte) error {
	var v struct {
		Miss   *float64 `json:"missInputTokens"`
		Hit    *float64 `json:"cacheReadTokens"`
		Output *float64 `json:"outputTokens"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	if v.Miss == nil || v.Hit == nil || v.Output == nil {
		return errors.New("missing token counters")
	}
	b.Miss, b.Hit, b.Output = *v.Miss, *v.Hit, *v.Output
	return nil
}
func valid(b *Bucket) bool {
	if b == nil {
		return false
	}
	for _, n := range []float64{b.Miss, b.Hit, b.Output} {
		if n < 0 || math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n > 9007199254740991 {
			return false
		}
	}
	return true
}
func add(a, b Bucket) Bucket { return Bucket{a.Miss + b.Miss, a.Hit + b.Hit, a.Output + b.Output} }

type Route struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Peak     *Bucket `json:"peak"`
	OffPeak  *Bucket `json:"offPeak"`
}
type Projection struct {
	Models []Route `json:"models"`
}

func Decode(raw json.RawMessage) *Projection {
	if !bytes.Contains(raw, []byte(`"billedUsage"`)) {
		return nil
	}
	var v struct {
		Billed *Projection `json:"billedUsage"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Billed == nil || v.Billed.Models == nil {
		return nil
	}
	for _, r := range v.Billed.Models {
		if !valid(r.Peak) || !valid(r.OffPeak) {
			return nil
		}
	}
	return v.Billed
}

type Source struct {
	ID, Parent, Origin, Name string
	Usage                    *Projection
}
type Row struct {
	Route
	Cents       float64
	Priced      bool
	Price       usage.ModelPrice
	PriceSource string
}
type Result struct {
	Rows                       []Row
	Cents                      float64
	Priced, Unpriced, Children int
	Partial, Available         bool
	Total                      Bucket
}

func Calculate(id string, sources []Source, settings usage.BillingSettings) Result {
	result := Result{}
	selected := map[string]bool{id: true}
	for changed := true; changed; {
		changed = false
		for _, s := range sources {
			if s.Origin == "subagent" && selected[s.Parent] && !selected[s.ID] {
				selected[s.ID] = true
				changed = true
			}
		}
	}
	seen := map[string]bool{}
	indexes := map[[2]string]int{}
	for _, s := range sources {
		if !selected[s.ID] || seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		if s.Usage == nil {
			result.Partial = true
			continue
		}
		result.Available = true
		if s.ID != id && len(s.Usage.Models) > 0 {
			result.Children++
		}
		for _, r := range s.Usage.Models {
			key := [2]string{r.Provider, r.Model}
			i, ok := indexes[key]
			if !ok {
				a, b := Bucket{}, Bucket{}
				i = len(result.Rows)
				indexes[key] = i
				result.Rows = append(result.Rows, Row{Route: Route{Provider: r.Provider, Model: r.Model, Peak: &a, OffPeak: &b}})
			}
			row := &result.Rows[i]
			*row.Peak = add(*row.Peak, *r.Peak)
			*row.OffPeak = add(*row.OffPeak, *r.OffPeak)
		}
	}
	if !seen[id] {
		result.Partial = true
	}
	for i := range result.Rows {
		r := &result.Rows[i]
		result.Total = add(result.Total, add(*r.Peak, *r.OffPeak))
		p, ok := usage.ResolveModelPrice(r.Provider, r.Model, settings.Prices)
		if !ok || !validRates(p.PriceRates) {
			result.Unpriced++
			continue
		}
		idle := p.PriceRates
		if !p.Flat {
			if p.Idle == nil {
				idle = usage.PriceRates{InputCacheMiss: p.InputCacheMiss / 2, InputCacheHit: p.InputCacheHit / 2, Output: p.Output / 2}
			}
			if p.Idle != nil {
				idle = *p.Idle
			}
		}
		cost := func(b Bucket, p usage.PriceRates) float64 {
			return b.Miss*math.Round(p.InputCacheMiss*1e6) + b.Hit*math.Round(p.InputCacheHit*1e6) + b.Output*math.Round(p.Output*1e6)
		}
		if !validRates(idle) {
			result.Unpriced++
			continue
		}
		r.Price = p
		r.Price.Idle = &idle
		r.PriceSource = "official"
		if _, custom := settings.Prices[r.Provider+"/"+r.Model]; custom {
			r.PriceSource = "custom"
		} else if _, custom := settings.Prices[r.Model]; custom {
			r.PriceSource = "custom"
		}
		n := cost(*r.Peak, p.PriceRates) + cost(*r.OffPeak, idle)
		if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 9007199254740991 {
			result.Unpriced++
			continue
		}
		r.Cents = math.Round(n / 1e10)
		r.Priced = true
		result.Priced++
		result.Cents += r.Cents
	}
	return result
}

// Phase is a display-only countdown; historical costs always use Host buckets.
func Phase(now time.Time) (bool, time.Time) {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	n := now.In(loc)
	peak := func(t time.Time) bool {
		h := t.Hour()
		return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday && ((h >= 9 && h < 12) || (h >= 14 && h < 18))
	}
	current := peak(n)
	next := n.Truncate(time.Hour).Add(time.Hour)
	for peak(next) == current {
		next = next.Add(time.Hour)
	}
	return current, next
}

func validRates(p usage.PriceRates) bool {
	for _, n := range []float64{p.InputCacheMiss, p.InputCacheHit, p.Output} {
		if n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
			return false
		}
	}
	return true
}
