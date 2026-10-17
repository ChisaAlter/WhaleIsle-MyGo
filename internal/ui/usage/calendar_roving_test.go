package usage

import (
	"github.com/egoist/mygo/ui"
	"testing"
	usagestore "whaleisle/internal/store/usage"
)

func TestCalendarRovingTab(t *testing.T) {
	s := usagestore.New()
	st := usagestore.StatsView{Overview: &usagestore.Overview{}}
	st.Overview.Days = []usagestore.DayUsage{{Date: "2026-10-01"}, {Date: "2026-10-02"}, {Date: "2026-10-03"}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Button(c, "Before")
			calendarUsageView(c, Model{Store: s}, st, nil, usagestore.BillingSettings{}, c.Theme().Text, c.Theme().TextMuted, c.Theme().Text, c.Theme().TextMuted)
			ui.Button(c, "After")
		})
	}, 1000, 900)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("2026-10-03 UTC") {
		t.Fatal("Tab should enter at last day")
	}
	tt.Key(0, ui.KeyUp)
	if !tt.Focused("2026-10-02 UTC") {
		t.Fatal("arrow did not move focus")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("After") {
		t.Fatal("Tab traversed another day")
	}
	tt.Key(ui.Shift, ui.KeyTab)
	if !tt.Focused("2026-10-02 UTC") {
		t.Fatal("return lost roving day")
	}
}
