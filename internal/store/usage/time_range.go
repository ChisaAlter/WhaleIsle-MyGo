package usage

import (
	"encoding/json"
	"errors"
	"sort"
	"time"
)

// DateRange uses inclusive UTC day keys, matching the overview's daily buckets.
type DateRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func (r DateRange) Active() bool { return r.Start != "" && r.End != "" }

type RangeSelection struct {
	Days   []DayUsage   `json:"days"`
	Totals Totals       `json:"totals"`
	Models []ModelUsage `json:"byModel"`
}

func (v StatsView) Selection() RangeSelection {
	out := RangeSelection{Days: []DayUsage{}, Models: []ModelUsage{}}
	if v.Overview == nil {
		return out
	}
	providers := map[string]string{}
	for _, m := range v.Overview.AllTime.ByModel {
		providers[m.Model] = m.Provider
	}
	models := map[string]Totals{}
	costs := map[string]*PhaseBuckets{}
	missingCosts := map[string]bool{}
	for _, d := range v.Overview.Days {
		if v.Range.Active() && (d.Date < v.Range.Start || d.Date > v.Range.End) {
			continue
		}
		out.Days = append(out.Days, d)
		out.Totals.Total += d.Total
		for name, t := range d.Models {
			x := models[name]
			addTotals(&x, t)
			models[name] = x
			phase := d.ModelCosts[name]
			if phase == nil || phase.Peak == nil || phase.OffPeak == nil {
				missingCosts[name] = true
			} else {
				if costs[name] == nil {
					costs[name] = &PhaseBuckets{Peak: &Totals{}, OffPeak: &Totals{}}
				}
				addTotals(costs[name].Peak, *phase.Peak)
				addTotals(costs[name].OffPeak, *phase.OffPeak)
			}
			out.Totals.Input += t.Input
			out.Totals.Output += t.Output
			out.Totals.CacheRead += t.CacheRead
			out.Totals.CacheWrite += t.CacheWrite
		}
	}
	for name, t := range models {
		if missingCosts[name] {
			costs[name] = nil
		}
		out.Models = append(out.Models, ModelUsage{Model: name, Provider: providers[name], Totals: t, Cost: costs[name]})
	}
	sort.Slice(out.Models, func(i, j int) bool {
		if out.Models[i].Total == out.Models[j].Total {
			return out.Models[i].Model < out.Models[j].Model
		}
		return out.Models[i].Total > out.Models[j].Total
	})
	return out
}
func addTotals(dst *Totals, src Totals) {
	dst.Input += src.Input
	dst.Output += src.Output
	dst.CacheRead += src.CacheRead
	dst.CacheWrite += src.CacheWrite
	dst.Total += src.Total
}
func (s *Store) SetStatsDateRange(start, end string) error {
	a, err := time.Parse("2006-01-02", start)
	if err != nil {
		return errors.New("Start date must be YYYY-MM-DD (UTC)")
	}
	b, err := time.Parse("2006-01-02", end)
	if err != nil || b.Before(a) {
		return errors.New("End date must be YYYY-MM-DD and on or after start (UTC)")
	}
	s.mu.Lock()
	var o Overview
	if json.Unmarshal(s.stats.Raw, &o) != nil || len(o.Days) == 0 {
		s.mu.Unlock()
		return errors.New("No daily coverage available")
	}
	// Every requested day must be present. Missing days are not treated as zero usage.
	available := map[string]bool{}
	for _, d := range o.Days {
		available[d.Date] = true
	}
	if int(b.Sub(a).Hours()/24)+1 > len(available) {
		s.mu.Unlock()
		return errors.New("Select dates within the available daily coverage")
	}
	for d := a; !d.After(b); d = d.AddDate(0, 0, 1) {
		if !available[d.Format("2006-01-02")] {
			s.mu.Unlock()
			return errors.New("Selected range includes unavailable daily coverage")
		}
	}
	s.ranking.modelLimit = 0
	s.stats.Range = DateRange{start, end}
	s.stats.RangeError = ""
	s.mu.Unlock()
	s.fire()
	return nil
}
func (s *Store) ClearStatsDateRange() {
	s.mu.Lock()
	s.ranking.modelLimit = 0
	s.stats.Range = DateRange{}
	s.stats.RangeStart = ""
	s.stats.RangeEnd = ""
	s.stats.RangeError = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetStatsRangeDraft(start, end string) {
	s.mu.Lock()
	s.stats.RangeStart = start
	s.stats.RangeEnd = end
	s.mu.Unlock()
}
func (s *Store) ApplyStatsRangeDraft() {
	v := s.Stats()
	err := s.SetStatsDateRange(v.RangeStart, v.RangeEnd)
	if err != nil {
		s.mu.Lock()
		s.stats.RangeError = err.Error()
		s.mu.Unlock()
		s.fire()
	}
}
func (s *Store) SelectStatsDay(day string, extend bool) {
	v := s.Stats()
	start, end := day, day
	if extend && v.Range.Active() {
		start = v.Range.Start
		end = v.Range.End
		if day < start {
			start = day
		} else {
			end = day
		}
	}
	s.SetStatsRangeDraft(start, end)
	s.ApplyStatsRangeDraft()
}

// filteredExport preserves the raw daily price buckets and coverage diagnostics.
// It deliberately omits all-time/session fields: the wire has no dated sessions.
func filteredExport(raw json.RawMessage, v StatsView) (json.RawMessage, error) {
	var src map[string]json.RawMessage
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, err
	}
	var days []json.RawMessage
	if err := json.Unmarshal(src["days"], &days); err != nil {
		return nil, err
	}
	kept := []json.RawMessage{}
	for _, rawDay := range days {
		var d DayUsage
		if err := json.Unmarshal(rawDay, &d); err != nil {
			return nil, err
		}
		if d.Date >= v.Range.Start && d.Date <= v.Range.End {
			kept = append(kept, rawDay)
		}
	}
	selection := v.Selection()
	return json.Marshal(map[string]any{"range": v.Range, "timezone": "UTC", "days": kept, "totals": selection.Totals, "byModel": selection.Models, "coverage": src["coverage"], "updatedAt": src["updatedAt"], "stale": src["stale"], "scope": "Selected available UTC days; session ranking unavailable for date ranges. Coverage diagnostics apply to the full scan."})
}

func (s *Store) SelectStatsRecentDays(n int) {
	s.SetStatsDays(n)
	v := s.Stats()
	if v.Overview == nil || len(v.Overview.Days) == 0 {
		return
	}
	days := append([]DayUsage(nil), v.Overview.Days...)
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	// The host's last day anchors presets, avoiding local timezone or clock skew.
	end, err := time.Parse("2006-01-02", days[len(days)-1].Date)
	if err != nil {
		return
	}
	s.SetStatsRangeDraft(end.AddDate(0, 0, -n+1).Format("2006-01-02"), end.Format("2006-01-02"))
	s.ApplyStatsRangeDraft()
}

// A refresh can move the rolling window. Never keep a now-partial selection.
func (s *Store) reconcileStatsRangeLocked() {
	r := s.stats.Range
	if !r.Active() {
		return
	}
	var o Overview
	if json.Unmarshal(s.stats.Raw, &o) != nil {
		return
	}
	available := map[string]bool{}
	for _, d := range o.Days {
		available[d.Date] = true
	}
	start, e1 := time.Parse("2006-01-02", r.Start)
	end, e2 := time.Parse("2006-01-02", r.End)
	valid := e1 == nil && e2 == nil
	for d := start; valid && !d.After(end); d = d.AddDate(0, 0, 1) {
		valid = available[d.Format("2006-01-02")]
	}
	if !valid {
		s.ranking.modelLimit = 0
		s.stats.Range = DateRange{}
		s.stats.RangeError = "Date selection cleared: refreshed daily coverage no longer contains the full range."
	}
}
