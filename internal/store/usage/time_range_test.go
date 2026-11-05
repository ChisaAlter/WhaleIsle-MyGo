package usage

import (
	"encoding/json"
	"strings"
	"testing"
)

// The inclusive filter belongs to the heatmap summary, not to export scope.
func TestStatsDateRangeSelectionExportAndClear(t *testing.T) {
	s := New()
	s.stats.Raw = json.RawMessage(`{"days":[{"date":"2026-10-12","total":2,"models":{"a":{"input":2,"total":2}}},{"date":"2026-10-13","total":3,"models":{"b":{"output":3,"total":3}}},{"date":"2026-10-14","total":5,"models":{"a":{"input":5,"total":5}}}],"allTime":{"totals":{"total":100},"byModel":[{"model":"a","provider":"p","total":97},{"model":"b","provider":"q","total":3}]},"topSessions":[{"id":"outside","totals":{"total":100}}],"coverage":{"timezone":"UTC","sessionsFailed":1},"updatedAt":1}`)
	if err := s.SetStatsDateRange("2026-10-13", "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	v := s.Stats()
	selected := v.Selection()
	if selected.Totals.Total != 8 || len(selected.Days) != 2 || selected.Models[0].Model != "a" || selected.Models[0].Total != 5 {
		t.Fatalf("bad selection: %+v", selected)
	}
	for _, format := range []string{"days", "json", "sessions"} {
		b, err := s.ExportStats(format)
		if err != nil {
			t.Fatal(err)
		}
		want := "2026-10-12"
		if format == "sessions" {
			want = "outside"
		}
		if !strings.Contains(string(b), want) {
			t.Fatalf("calendar selection narrowed %s export: %s", format, b)
		}
	}
	if err := s.SetStatsDateRange("2026-10-14", "2026-10-12"); err == nil {
		t.Fatal("reversed date accepted")
	}
	if s.Stats().Range.Start != "2026-10-13" {
		t.Fatal("invalid input replaced selection")
	}
	if err := s.SetStatsDateRange("2026-10-01", "2026-10-14"); err == nil {
		t.Fatal("out-of-window range accepted")
	}
	s.ClearStatsDateRange()
	if s.Stats().Range.Active() {
		t.Fatal("selection not cleared")
	}
	b, err := s.ExportStats("sessions")
	if err != nil || !strings.Contains(string(b), "outside") {
		t.Fatalf("clear did not restore export: %s %v", b, err)
	}
}
