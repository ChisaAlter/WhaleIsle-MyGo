package usage

import (
	"fmt"
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
	if !tt.HasText(i18n.T(i18n.KeySurfacesUsageUpdated)) {
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
