package usage

import (
	"testing"
	"time"
	usagestore "whaleisle/internal/store/usage"
)

func TestCalendarMonthLabelsPartialLeadingMonth(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-09-28")
	end := start.AddDate(0, 0, 13)
	days := map[string]usagestore.DayUsage{}
	for i := 0; i < 14; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		days[d] = usagestore.DayUsage{Date: d}
	}
	labels := calendarMonthLabels(start, end, days)
	if len(labels) != 1 || labels["2026-10-05"] != "2026-10" {
		t.Fatalf("overlapping leading marker: %v", labels)
	}
	delete(days, "2026-09-28")
	delete(days, "2026-09-29")
	delete(days, "2026-09-30")
	labels = calendarMonthLabels(start, end, days)
	if labels["2026-09-28"] != "2026-10" {
		t.Fatalf("marker used padding instead of first actual day: %v", labels)
	}
}
