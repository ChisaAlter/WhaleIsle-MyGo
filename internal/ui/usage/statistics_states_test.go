package usage

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestEmbeddedUsageInitialLoadingErrorAndEmptyStates(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	for _, state := range []string{"loading", "error", "empty"} {
		t.Run(state, func(t *testing.T) {
			s := usageFixture(t, state)
			tt := ui.NewTester(func(c *ui.Context) { ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: s, Embedded: true}) }) }, 680, 1000)
			text := strings.Join(tt.Texts(), "\n")
			if state == "loading" && !strings.Contains(text, "Reading session usage statistics") {
				t.Fatal(text)
			}
			if state == "error" && !strings.Contains(text, s.Stats().Err) {
				t.Fatal("initial error hidden", text)
			}
			if state == "empty" && !strings.Contains(text, "No usage recorded yet") {
				t.Fatal(text)
			}
			if strings.Contains(text, "Date range (UTC") || strings.Contains(text, "Export json") {
				t.Fatal("empty/loading/error rendered data-only controls", text)
			}
		})
	}
}

func TestEmbeddedUsageRangeSharesDimensionsAndExportScope(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	s := usageFixture(t, "data")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: s, Embedded: true, OnExportStats: func(string) {}}) })
	}, 850, 2200)
	if !tt.HasText("Model share (all time)") || !tt.HasText("Daily model breakdown (recent)") || !tt.HasText("Export sessions") {
		t.Fatal("all-time/recent dimensions missing")
	}
	s.SelectStatsDay("2026-10-14", false)
	tt.Frame()
	if !tt.HasText("Model share (selected dates)") || !tt.HasText("Daily model breakdown (selected dates)") || tt.HasText("Export sessions") {
		t.Fatal("selection was not shared across dimensions and exports")
	}
	s.SetStatsRangeDraft("2026-10-15", "2026-10-14")
	s.ApplyStatsRangeDraft()
	tt.Frame()
	if s.Stats().RangeError == "" || !tt.HasText(s.Stats().RangeError) || !s.Stats().Range.Active() {
		t.Fatal("invalid range silently replaced accepted selection")
	}
	if err := tt.Click("7 days"); err != nil {
		t.Fatal(err)
	}
	if s.Stats().Days != 7 || s.Stats().RangeError == "" {
		t.Fatal("recent preset silently filled unavailable daily coverage")
	}
	if err := tt.Click("Clear"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if s.Stats().Range.Active() || !tt.HasText("Model share (all time)") || !tt.HasText("Export sessions") {
		t.Fatal("cleared range retained selected-date export scope")
	}
}
