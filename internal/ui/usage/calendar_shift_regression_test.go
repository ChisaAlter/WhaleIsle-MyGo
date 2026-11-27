package usage

import (
	"testing"

	"github.com/egoist/mygo/ui"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

func TestUsageCalendarHoverZeroVerticalShift(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })

	s := usagestore.New()
	st := usagestore.StatsView{Overview: &usagestore.Overview{}}
	st.Overview.Days = []usagestore.DayUsage{
		{Date: "2026-10-01", Total: 1200},
		{Date: "2026-10-02", Total: 3400},
		{Date: "2026-10-03", Total: 0},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			calendarUsageView(c, Model{Store: s}, st, nil, usagestore.BillingSettings{})
			ui.Text(c, "Below calendar sentinel")
		})
	}, 900, 600)

	lessBefore, ok1 := tt.Find("Less")
	belowBefore, ok2 := tt.Find("Below calendar sentinel")
	dayCell, ok3 := tt.Find("2026-10-02 UTC")
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("missing calendar elements: less=%v below=%v day=%v texts=%v", ok1, ok2, ok3, tt.Texts())
	}

	tt.Move(dayCell.X+dayCell.W/2, dayCell.Y+dayCell.H/2)
	tt.Frame()

	lessAfter, _ := tt.Find("Less")
	belowAfter, _ := tt.Find("Below calendar sentinel")
	if lessAfter != lessBefore || belowAfter != belowBefore {
		t.Fatalf("hovering calendar day shifted layout: less %+v -> %+v, below %+v -> %+v",
			lessBefore, lessAfter, belowBefore, belowAfter)
	}
}
