package usage

import (
	"encoding/json"
	"testing"
)

func TestCalendarShiftPickShrinksEnd(t *testing.T) {
	s := New()
	s.stats.Raw = json.RawMessage(`{"days":[{"date":"2026-10-01"},{"date":"2026-10-02"},{"date":"2026-10-03"}]}`)
	s.SelectStatsDay("2026-10-01", false)
	s.SelectStatsDay("2026-10-03", true)
	s.SelectStatsDay("2026-10-02", true)
	if v := s.Stats(); v.Range.Start != "2026-10-01" || v.Range.End != "2026-10-02" {
		t.Fatalf("shift should replace end: %+v", v.Range)
	}
	s.SelectStatsDay("2026-10-02", false)
	s.SelectStatsDay("2026-10-03", true)
	s.SelectStatsDay("2026-10-01", true)
	if v := s.Stats(); v.Range.Start != "2026-10-01" || v.Range.End != "2026-10-03" {
		t.Fatalf("earlier shift should preserve end: %+v", v.Range)
	}
}
