package usage

import (
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Synthetic data only. Removing official fallback, zero override membership,
// current-window model rows, or calendar-only selection breaks this fixture.
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
	row.Input, row.Output = 3e6, 2e6
	s := New()
	s.billing.Loaded = true
	s.billing.Settings = BillingSettings{Prices: map[string]ModelPrice{}, PeakValleyEnabled: true}
	overview := Overview{Days: []DayUsage{{Date: "2026-10-13", Total: 5e6, Models: map[string]Totals{row.Model: row.Totals}, ModelCosts: map[string]*PhaseBuckets{row.Model: phases}}, {Date: "2026-10-14", Total: 1, Models: map[string]Totals{"=unknown()": {Input: 1, Total: 1}}}}, ByModel: []ModelUsage{row, {Model: "=unknown()", Provider: "route", Totals: Totals{Input: 1, Total: 1}}}, UpdatedAt: 1, TopSessions: []SessionUsage{}}
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
	if len(rows) != 3 || !reflect.DeepEqual(rows[0], []string{"model", "provider", "total", "input", "output", "cacheRead", "cacheWrite", "costPeakCents", "costOffPeakCents", "costTotalCents"}) || rows[1][0] != row.Model || rows[1][1] != "route" || rows[1][7] != "1510" || rows[1][8] != "150" || rows[1][9] != "1660" || rows[2][0] != "'=unknown()" || rows[2][9] != "" {
		t.Fatalf("recent model CSV: %v", rows)
	}
	days := readCSV("days")
	if len(days) != 3 || !reflect.DeepEqual(days[0], []string{"date", "total", "input", "output", "cacheRead", "cacheWrite", "costPeakCents", "costOffPeakCents", "costTotalCents"}) || days[1][8] != "1660" || days[2][8] != "" {
		t.Fatalf("daily costs: %v", days)
	}
	if err := s.SetStatsDateRange("2026-10-13", "2026-10-13"); err != nil {
		t.Fatal(err)
	}
	selected := readCSV("models")
	if !reflect.DeepEqual(selected, rows) {
		t.Fatalf("calendar selection changed model export: %v", selected)
	}
	if !reflect.DeepEqual(readCSV("days"), days) {
		t.Fatal("calendar selection changed daily export")
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
	if got := s.Stats().Selection(); len(got.Models) != 1 || got.Models[0].Cost != nil {
		t.Fatalf("missing dated periods invented summary estimate: %+v", got)
	}
	if got := readCSV("models"); !reflect.DeepEqual(got, rows) {
		t.Fatalf("daily coverage changed recent model export: %v", got)
	}
}

func TestCurrentModelPriceRestoresDefaultWithoutSubmittingOtherBuffers(t *testing.T) {
	s := New()
	s.billing.Loaded = true
	s.billing.Settings = BillingSettings{PeakValleyEnabled: true, Prices: map[string]ModelPrice{"deepseek-v4-flash": {PriceRates: PriceRates{9, 9, 9}}, "other/model": {PriceRates: PriceRates{1, 2, 3}, Flat: true}}}
	s.billing.Drafts = priceDrafts(s.billing.Settings.Prices)
	s.EditPrice("other/model", PriceDraft{Peak: [3]string{"5", "5", "5"}, Flat: true})
	price, ok := ResolveModelPrice("deepseek", "deepseek-v4-flash", nil)
	if !ok || price.Idle == nil {
		t.Fatal("missing pinned price fixture")
	}
	draft := PriceDraft{Peak: draftRates(price.PriceRates), Idle: draftRates(*price.Idle)}
	if !s.EditModelPrice("deepseek/deepseek-v4-flash", draft) {
		t.Fatal(s.Billing().Err)
	}
	v := s.Billing()
	if _, exists := v.Drafts["deepseek-v4-flash"]; exists {
		t.Fatal("default restore kept the legacy bare override")
	}
	if _, exists := v.Drafts["deepseek/deepseek-v4-flash"]; exists {
		t.Fatal("default restore persisted a redundant composite override")
	}
	if v.Drafts["other/model"].Peak != [3]string{"1", "2", "3"} {
		t.Fatal("current model commit submitted another model's uncommitted buffer")
	}
}
