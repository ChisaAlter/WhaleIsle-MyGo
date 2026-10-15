package usage

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

// Synthetic data only. Removing official fallback, zero override membership,
// current-window model rows, or selected daily cost buckets breaks this fixture.
func TestUsageParityPricingExportFixture(t *testing.T) {
	settings := BillingSettings{Prices: map[string]ModelPrice{}, PeakValleyEnabled: true}
	phases := &PhaseBuckets{Peak: &Totals{Input: 1e6, CacheWrite: 1e6, CacheRead: 1e6, Output: 1e6}, OffPeak: &Totals{Input: 1e6}}
	row := ModelUsage{Provider: "route", Model: "DEEPSEEK-V4-FLASH", Cost: phases}
	for _, tc := range []struct {
		model string
		want  float64
	}{{"DEEPSEEK-V4-FLASH", 1660}, {"deepseek-v4-pro", 4980}, {"deepseek-v4-flash-vision-exp", 1660}} {
		row.Model = tc.model
		if got, ok := EstimateModel(row, settings); !ok || got != tc.want {
			t.Fatalf("official %s: %v %v", tc.model, got, ok)
		}
	}
	row.Model = "DEEPSEEK-V4-FLASH"
	settings.PeakValleyEnabled = false
	if got, ok := EstimateModel(row, settings); !ok || got != 1810 {
		t.Fatalf("global flat: %v %v", got, ok)
	}
	settings.PeakValleyEnabled = true
	settings.Prices[row.Model] = ModelPrice{PriceRates: PriceRates{1, 2, 4}}
	settings.Prices["route/"+row.Model] = ModelPrice{} // Intentional zero must override both fallbacks.
	if got, ok := EstimateModel(row, settings); !ok || got != 0 {
		t.Fatalf("zero override: %v %v", got, ok)
	}
	delete(settings.Prices, "route/"+row.Model)
	if got, ok := EstimateModel(row, settings); !ok || got != 1000 {
		t.Fatalf("bare override: %v %v", got, ok)
	}
	settings.Prices[row.Model] = ModelPrice{PriceRates: PriceRates{1, 2, 4}, Idle: &PriceRates{3, 6, 8}}
	if got, ok := EstimateModel(row, settings); !ok || got != 1500 {
		t.Fatalf("explicit idle: %v %v", got, ok)
	}
	settings.Prices[row.Model] = ModelPrice{PriceRates: PriceRates{1, 2, 4}, Flat: true}
	if got, ok := EstimateModel(row, settings); !ok || got != 1100 {
		t.Fatalf("per-model flat: %v %v", got, ok)
	}
	row.Model = "not-in-table"
	if _, ok := EstimateModel(row, settings); ok {
		t.Fatal("unknown model invented price")
	}
	row.Model = "deepseek-v4-pro"
	row.Cost = &PhaseBuckets{Peak: &Totals{}}
	if _, ok := EstimateModel(row, settings); ok {
		t.Fatal("missing period became free")
	}
	row.Model = "deepseek-v4-flash"
	row.Cost = phases
	row.Total = 5e6
	s := New()
	s.billing.Loaded = true
	s.billing.Settings = BillingSettings{Prices: map[string]ModelPrice{}, PeakValleyEnabled: true}
	overview := Overview{Days: []DayUsage{{Date: "2026-10-13", Total: 5e6, Models: map[string]Totals{row.Model: {Total: 5e6}}, ModelCosts: map[string]*PhaseBuckets{row.Model: phases}}, {Date: "2026-10-14", Total: 1, Models: map[string]Totals{"=unknown()": {Input: 1, Total: 1}}}}, ByModel: []ModelUsage{row, {Model: "=unknown()", Provider: "route", Totals: Totals{Total: 1}}}, UpdatedAt: 1, TopSessions: []SessionUsage{}}
	overview.AllTime.ByModel = []ModelUsage{{Model: row.Model, Provider: "route"}, {Model: "historical-only", Totals: Totals{Total: 99}}}
	overview.Coverage.Timezone = "UTC"
	raw, err := json.Marshal(overview)
	if err != nil {
		t.Fatal(err)
	}
	s.stats.Raw = raw
	readCSV := func(kind string) [][]string {
		t.Helper()
		b, err := s.ExportStats(kind)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(b), "\xef\xbb\xbf"))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	rows := readCSV("models")
	if len(rows) != 3 || rows[1][1] != row.Model || rows[1][8] != "1510" || rows[1][9] != "150" || rows[1][10] != "1660" || rows[1][11] != "CNY" || rows[2][1] != "'=unknown()" || rows[2][10] != "" {
		t.Fatalf("recent model CSV: %v", rows)
	}
	days := readCSV("days")
	if len(days) != 3 || days[1][10] != "1660" || days[2][10] != "" || days[2][13] != "1" {
		t.Fatalf("daily costs: %v", days)
	}
	if err := s.SetStatsDateRange("2026-10-13", "2026-10-13"); err != nil {
		t.Fatal(err)
	}
	selected := readCSV("models")
	if len(selected) != 2 || selected[1][10] != "1660" {
		t.Fatalf("selected buckets lost: %v", selected)
	}
	if len(readCSV("days")) != 2 {
		t.Fatal("daily export ignored selection")
	}
	// If any selected day lacks this model's period buckets, the aggregated model
	// estimate stays unknown instead of reporting a misleading partial full cost.
	overview.Days[1].Models = map[string]Totals{row.Model: {Input: 1, Total: 1}}
	raw, _ = json.Marshal(overview)
	s.mu.Lock()
	s.stats.Raw = raw
	s.mu.Unlock()
	if err := s.SetStatsDateRange("2026-10-13", "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	if got := readCSV("models"); got[1][10] != "" {
		t.Fatalf("missing dated periods invented estimate: %v", got)
	}
}
