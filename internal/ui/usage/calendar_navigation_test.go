package usage

import (
	"github.com/egoist/mygo/ui"
	"testing"
	usagestore "whaleisle/internal/store/usage"
	fonttheme "whaleisle/internal/ui/theme"
)

func TestCalendarFocusNavigation(t *testing.T) {
	days := []usagestore.DayUsage{{Date: "2026-10-01"}, {Date: "2026-10-02"}, {Date: "2026-10-08"}, {Date: "2026-10-09"}}
	for _, tt := range []struct {
		from string
		key  ui.Key
		want string
	}{
		{"2026-10-01", ui.KeyDown, "2026-10-02"}, {"2026-10-02", ui.KeyUp, "2026-10-01"},
		{"2026-10-01", ui.KeyRight, "2026-10-08"}, {"2026-10-09", ui.KeyLeft, "2026-10-02"},
		{"2026-10-01", ui.KeyUp, ""}, {"2026-10-02", ui.KeyDown, ""},
		{"2026-10-08", ui.KeyHome, "2026-10-01"}, {"2026-10-02", ui.KeyEnd, "2026-10-09"},
	} {
		if got := calendarFocusTarget(tt.from, tt.key, days); got != tt.want {
			t.Fatalf("%s key %v: %q != %q", tt.from, tt.key, got, tt.want)
		}
	}
}

func TestPriceDialogDismissPreservesDraft(t *testing.T) {
	open := false
	draft := "1.25"
	tt := ui.NewTester(func(c *ui.Context) {
		if ui.Button(c, "Prices").Clicked() {
			open = true
		}
		pricingDialog(c, &open, fonttheme.Resolve("").Active(false), func() { ui.TextInput(c, &draft).Label("Price draft") })
	}, 1000, 900)
	tt.Click("Prices")
	tt.Key(0, ui.KeyEscape)
	if open || !tt.Focused("Prices") || draft != "1.25" {
		t.Fatal("dismissal lost draft or opener focus")
	}
	tt.Click("Prices")
	if !open {
		t.Fatal("dialog did not reopen")
	}
}
