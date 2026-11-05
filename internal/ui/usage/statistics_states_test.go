package usage

import (
	"bytes"
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
			if state == "loading" && !strings.Contains(text, "Loading usage") {
				t.Fatal(text)
			}
			if state == "error" && !strings.Contains(text, s.Stats().Err) {
				t.Fatal("initial error hidden", text)
			}
			if state == "empty" && !strings.Contains(text, "No usage in this range.") {
				t.Fatal(text)
			}
			if strings.Contains(text, "Date range (UTC") || strings.Contains(text, "Export json") {
				t.Fatal("empty/loading/error rendered data-only controls", text)
			}
		})
	}
}

func TestEmbeddedUsageCalendarSelectionKeepsOtherDimensionsAndExports(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	s := usageFixture(t, "data")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { View(c, Model{Store: s, Embedded: true, OnExportStats: func(string) {}}) })
	}, 850, 3600)
	if !tt.HasText("Model share (all time)") || !tt.HasText("Daily model breakdown (recent)") || !tt.HasText("Session ranking") || !tt.HasText("Export") {
		t.Fatal("all-time/recent dimensions missing:" + "\n" + strings.Join(tt.Texts(), "\n"))
	}
	before, err := s.ExportStats("json")
	if err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("2026-10-14 UTC"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !s.Stats().Range.Active() || !tt.HasText("Model share (all time)") || !tt.HasText("Daily model breakdown (recent)") || !tt.HasText("Session ranking") {
		t.Fatal("calendar selection changed independent dashboard dimensions")
	}
	s.SetStatsRangeDraft("2026-10-15", "2026-10-14")
	s.ApplyStatsRangeDraft()
	tt.Frame()
	if s.Stats().RangeError == "" || !s.Stats().Range.Active() {
		t.Fatal("invalid range silently replaced accepted selection")
	}
	if err := tt.Click("Last 7 days"); err != nil {
		t.Fatal(err)
	}
	if s.Stats().Days != 7 || !s.Stats().Range.Active() {
		t.Fatal("recent chart window replaced the calendar selection")
	}
	after, err := s.ExportStats("json")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("calendar/chart controls changed JSON export", err)
	}
	if err := tt.Click("Clear selection"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if s.Stats().Range.Active() || !tt.HasText("Model share (all time)") || !tt.HasText("Export") {
		t.Fatal("calendar selection was not cleared independently")
	}
}
