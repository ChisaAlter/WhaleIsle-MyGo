package schedule

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	schedstore "whaleisle/internal/store/schedule"
)

func TestB27CronBuilderRoundTrip(t *testing.T) {
	for _, expr := range []string{"*/15 * * * *", "5 */2 * * *", "30 9 * * *", "0 9 * * 1-5", "0 9 1,15,31 * *", "0 9 * * 0-6", "0 9 1-31 * *"} {
		parsed, err := parseTaskCron(expr)
		if err != nil {
			t.Fatal(expr, err)
		}
		shape, ok := parsed.shape()
		if !ok {
			t.Fatal("unrecognized", expr)
		}
		again, err := parseTaskCron(shape.expression())
		if err != nil {
			t.Fatal(err)
		}
		next, ok := again.shape()
		if !ok || next.kind != shape.kind || next.expression() != shape.expression() {
			t.Fatalf("round trip %s: %+v", expr, next)
		}
	}
	for _, expr := range []string{"60 * * * *", "0 24 * * *", "0 9 0 * *", "0 9 * 13 *", "0 9 * * MON", "1/2 * * * *", "*/0 * * * *", "5-1 * * * *", "0 9 * * 8", "0 9 * * 1,"} {
		if _, err := parseTaskCron(expr); err == nil {
			t.Fatal("accepted", expr)
		}
	}
	for _, expr := range []string{"0 9 1 * 1", "0 9 * 2 *", "0,30 9 * * *"} {
		p, err := parseTaskCron(expr)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := p.shape(); ok {
			t.Fatal("raw must stay raw", expr)
		}
	}
	union, _ := parseTaskCron("0 9 1 * 1")
	intersection, _ := parseTaskCron("0 9 */2 * 1")
	if !strings.Contains(union.preview(), " OR ") || !strings.Contains(intersection.preview(), " AND ") {
		t.Fatal("day semantics lost")
	}
}
func TestB27CalendarClockAndWire(t *testing.T) {
	month := time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)
	cells := calendarCells(month)
	if len(cells) != 35 || cells[1] != 1 || cells[29] != 29 {
		t.Fatalf("leap calendar %v", cells)
	}
	d := newTaskDraft(schedstore.Entry{Raw: json.RawMessage(`{"kind":"at","scheduledAt":"2026-12-01T09:12:13+08:00"}`)})
	if err := d.pickWallClock("2028-02-29", "10:20:30", "Asia/Shanghai"); err != nil {
		t.Fatal(err)
	}
	raw, err := d.change()
	if err != nil || !strings.Contains(string(raw), "2028-02-29T10:20:30+08:00") {
		t.Fatalf("%s %v", raw, err)
	}
	if err := d.pickWallClock("2028-02-30", "10:20:30", "Asia/Shanghai"); err == nil {
		t.Fatal("invalid date")
	}
	if err := d.pickWallClock("2026-03-08", "02:30:00", "America/New_York"); err == nil {
		t.Fatal("DST nonexistent clock")
	}
	if clockPick("09:12:13.123", 1, "45") != "09:45:13" {
		t.Fatal("clock columns")
	}
	d.Kind = "cron"
	d.Expression = "60 * * * *"
	d.Zone = "UTC"
	if _, err := d.change(); err == nil {
		t.Fatal("invalid cron wire")
	}
	d.Expression = "0 9 * * 1-5"
	raw, err = d.change()
	if err != nil || !strings.Contains(string(raw), `"expression":"0 9 * * 1-5"`) {
		t.Fatalf("%s %v", raw, err)
	}
}
func TestB27RecentZones(t *testing.T) {
	t.Setenv("WHALEISLE_USER_DATA", t.TempDir())
	r := loadRecentZones("Asia/Shanghai")
	if strings.Join(r, ",") != "Asia/Shanghai,UTC" {
		t.Fatal(r)
	}
	for _, z := range []string{"Europe/London", "America/New_York", "Asia/Tokyo", "Europe/Paris", "UTC", "Asia/Tokyo"} {
		r = rememberZone(r, z)
	}
	got := loadRecentZones("UTC")
	if len(got) != 5 || got[0] != "Asia/Tokyo" || got[1] != "UTC" {
		t.Fatal(got)
	}
	if len(toggleNumber([]int{1}, 1)) != 1 {
		t.Fatal("last selected day removed")
	}
}
