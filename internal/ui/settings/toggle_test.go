package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestSettingsToggleIsTheHarnessSwitch pins the 44×24 switch: the size the
// installed build uses, and a click that flips the caller's value once.
func TestSettingsToggleIsTheHarnessSwitch(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	value := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() {
			if settingsToggle(c, "Probe switch", &value, false).Clicked() {
				value = !value
			}
		})
	}, 320, 120)
	rect, ok := tt.Find("Probe switch")
	if !ok {
		t.Fatalf("switch is not reachable: %v", tt.Texts())
	}
	if rect.W != toggleWidth || rect.H != toggleHeight {
		t.Fatalf("switch is %v×%v, want %v×%v", rect.W, rect.H, toggleWidth, toggleHeight)
	}
	if err := tt.Click("Probe switch"); err != nil {
		t.Fatalf("switch is not clickable: %v", err)
	}
	if !value {
		t.Fatal("clicking the switch did not flip the value")
	}
}

// TestBodyToggleRowBadgeKeepsTheRowGeometryAndBadge covers the beta row: the badge
// sits next to the title, the row keeps the 64 DIP height and the switch toggles.
func TestBodyToggleRowBadgeKeepsTheRowGeometryAndBadge(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	value := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Children(func() {
			bodyToggleRowBadge(c, "probe-badge", "Bots", "Beta", "Turns on the sidebar bot contact.", &value, false, c.Theme().Text, c.Theme().TextMuted, nil)
			bodyToggleRowBadge(c, "probe-after", "Second row", "", "Keeps the pitch honest.", new(bool), false, c.Theme().Text, c.Theme().TextMuted, nil)
		})
	}, 720, 260)
	for _, want := range []string{"Bots", "Beta", "Turns on the sidebar bot contact."} {
		if !tt.HasText(want) {
			t.Fatalf("badge row lost %q: %v", want, tt.Texts())
		}
	}
	first, _ := tt.Find("Bots")
	second, _ := tt.Find("Second row")
	if pitch := second.Y - first.Y; pitch < 64 {
		t.Fatalf("badge row pitch = %v, want at least 64", pitch)
	}
	if err := tt.Click("probe-badge"); err != nil {
		t.Fatalf("badge row switch is not clickable: %v", err)
	}
	if !value {
		t.Fatal("badge row switch did not flip its value")
	}
}

// TestSettingsSegmentedReportsThePick covers the page-level segmented control:
// both captions paint and a click reports the segment index.
func TestSettingsSegmentedReportsThePick(t *testing.T) {
	picked := -1
	active := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() {
			_, choice := settingsSegmented(c, "probe-tabs", []string{"Chat and abilities", "Appearance and behavior"}, active, false)
			if choice >= 0 {
				picked = choice
			}
		})
	}, 640, 120)
	for _, want := range []string{"Chat and abilities", "Appearance and behavior"} {
		if !tt.HasText(want) {
			t.Fatalf("segment %q missing: %v", want, tt.Texts())
		}
	}
	if err := tt.Click("Appearance and behavior"); err != nil {
		t.Fatalf("second segment is not clickable: %v", err)
	}
	if picked != 1 {
		t.Fatalf("segmented reported %d, want 1", picked)
	}
}
