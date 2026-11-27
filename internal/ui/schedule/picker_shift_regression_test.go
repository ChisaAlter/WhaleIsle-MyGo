package schedule

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/i18n"
)

func TestDateAndClockPickersReserveBorderWithoutShift(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	d := newTaskDraft(schedstore.Entry{Raw: json.RawMessage(`{"kind":"at","scheduledAt":"2026-12-01T09:15:00Z"}`)})
	d.picker.calendarOpen = true
	d.picker.month = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	p := &Panel{recentZones: []string{"UTC"}}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(12).Children(func() { p.datePicker(c, d) })
	}, 360, 520)
	base := time.Unix(10000, 0)
	tt.SetTime(base.Add(300 * time.Millisecond))

	day10Before, ok := tt.Find("10")
	if !ok {
		t.Fatalf("calendar day 10 missing: %v", tt.Texts())
	}
	if err := tt.Click("2026-12-10"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	day10After, _ := tt.Find("10")
	if day10After != day10Before {
		t.Fatalf("selecting calendar day shifted cell text: %+v -> %+v", day10Before, day10After)
	}

	picked := "01:00"
	clockState := &scheduleClockPicker{open: true}
	ttClock := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(12).Children(func() {
			clockPicker(c, clockState, picked, false, func(v string) { picked = v })
		})
	}, 360, 520)
	ttClock.SetTime(base.Add(600 * time.Millisecond))
	h02Before, ok := ttClock.Find("Hour 02")
	if !ok || h02Before.W <= 0 || h02Before.H <= 0 {
		t.Fatalf("clock Hour 02 missing or collapsed: ok=%v rect=%+v texts=%v", ok, h02Before, ttClock.Texts())
	}
	if err := ttClock.Click("Hour 02"); err != nil {
		t.Fatal(err)
	}
	ttClock.Frame()
	if picked != "02:00:00" {
		t.Fatalf("clicking Hour 02 picked %q, want 02:00:00", picked)
	}
	h02After, ok := ttClock.Find("Hour 02")
	if !ok || h02After != h02Before {
		t.Fatalf("selecting clock option shifted cell: ok=%v %+v -> %+v", ok, h02Before, h02After)
	}
}
