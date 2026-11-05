package usage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func usageDashboardTester(t *testing.T, state string) *ui.Tester {
	t.Helper()
	s := usageFixture(t, state)
	return ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Store: s, Embedded: true, OnExportStats: func(string) {}})
		})
	}, 850, 3600)
}

// TestEmbeddedUsageEmptyStateHasNoDashboard keeps the empty page honest: the
// empty card is the whole answer, with no KPI grid, calendar or diagnostics
// rendered behind it.
func TestEmbeddedUsageEmptyStateHasNoDashboard(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	tt := usageDashboardTester(t, "empty")
	if !tt.HasText(i18n.T(i18n.KeySurfacesUsageEmpty)) {
		t.Fatalf("empty state missing its card: %v", tt.Texts())
	}
	for _, dashboard := range []string{
		i18n.T(i18n.KeySurfacesUsageTotal),
		i18n.T(i18n.KeySurfacesUsageCalendar),
		i18n.T(i18n.KeySurfacesUsageDiagnostic),
	} {
		if tt.HasText(dashboard) {
			t.Fatalf("empty state still renders %q: %v", dashboard, tt.Texts())
		}
	}
}

// TestRangeChipsSwitchTheWindow covers the toolbar: the recent-day choices set
// the window the dashboard reads.
func TestRangeChipsSwitchTheWindow(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	s := usageFixture(t, "data")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Store: s, Embedded: true, OnExportStats: func(string) {}})
		})
	}, 850, 3600)
	window := fmt.Sprintf(i18n.T(i18n.KeySurfacesUsageDays), 14)
	if err := tt.Click(window); err != nil {
		t.Fatalf("14-day chip is not reachable: %v", err)
	}
	if got := s.Stats().Days; got != 14 {
		t.Fatalf("chip did not switch the window: %d", got)
	}
	tt.Frame()
	if !tt.HasText("Updated at 09:26 · UTC") {
		t.Fatalf("toolbar lost its update stamp: %v", tt.Texts())
	}
}

// TestDiagnosticsStayFolded keeps the scan counters off the dashboard until the
// reader asks for them.
func TestDiagnosticsStayFolded(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	tt := usageDashboardTester(t, "data")
	if tt.HasText("Coverage:") {
		t.Fatalf("diagnostics printed without being opened: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesUsageDiagnostic)); err != nil {
		t.Fatalf("diagnostics disclosure is not reachable: %v", err)
	}
	tt.Frame()
	if !tt.HasText("Coverage:") {
		t.Fatalf("diagnostics did not open: %v", tt.Texts())
	}
}

// Recent-window controls keep independent all-time and ranking sections visible.
func TestRangeSwitchKeepsEverySectionConsistent(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	s := usageFixture(t, "data")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Store: s, Embedded: true, OnExportStats: func(string) {}})
		})
	}, 850, 3600)
	sections := []string{
		i18n.T(i18n.KeySurfacesUsageTotal),
		"Usage calendar",
		"Model share",
		"Daily model breakdown (recent)",
		"Project ranking (session working directories)",
		i18n.T(i18n.KeySurfacesUsageRanking),
	}
	for _, window := range []int{14, 30, 7} {
		chip := fmt.Sprintf(i18n.T(i18n.KeySurfacesUsageDays), window)
		if err := tt.Click(chip); err != nil {
			t.Fatalf("%d-day chip is not reachable: %v", window, err)
		}
		tt.Frame()
		if got := s.Stats().Days; got != window {
			t.Fatalf("window = %d, want %d", got, window)
		}
		for _, section := range sections {
			if !tt.HasText(section) {
				t.Fatalf("%d-day window dropped %q: %v", window, section, tt.Texts())
			}
		}
	}
}

// TestUsageStateCopyFollowsTheLocale keeps the empty and failure pages translated
// instead of printing English into a Chinese window (or the reverse).
func TestUsageStateCopyFollowsTheLocale(t *testing.T) {
	render := func(state string, locale string) string {
		i18n.SetLocale(locale)
		t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
		tt := usageDashboardTester(t, state)
		return strings.Join(tt.Texts(), "|")
	}
	for _, state := range []string{"empty", "error"} {
		en, zh := render(state, i18n.LocaleEN), render(state, i18n.LocaleZH)
		if en == "" || zh == "" {
			t.Fatalf("%s state rendered nothing", state)
		}
		if state == "empty" {
			if !strings.Contains(en, "No usage in this range.") || !strings.Contains(zh, "此范围暂无用量。") {
				t.Fatalf("%s state lost its copy: en=%s zh=%s", state, en, zh)
			}
		}
		if state == "error" {
			if !strings.Contains(en, "Try again later") || !strings.Contains(zh, "稍后重试") {
				t.Fatalf("%s state is not localised: en=%s zh=%s", state, en, zh)
			}
		}
	}
}
