package market

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

func assertMarketCursors(t *testing.T, viewName string, tt *ui.Tester) {
	t.Helper()
	for _, b := range tt.Interactives() {
		if slices.Contains(b.Kinds, "text") {
			if b.Cursor != ui.CursorText && b.Cursor != ui.CursorDefault && b.Cursor != ui.CursorNotAllowed {
				t.Errorf("[%s] text input %s has unexpected cursor %v", viewName, b.String(), b.Cursor)
			}
			continue
		}
		if slices.Contains(b.Kinds, "click") {
			if b.Cursor != ui.CursorPointer && b.Cursor != ui.CursorDefault && b.Cursor != ui.CursorNotAllowed {
				t.Errorf("[%s] clickable %s has unexpected cursor %v", viewName, b.String(), b.Cursor)
			}
		}
	}
}

func TestMarketParityAndCursorAudit(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			st := marketstore.New()
			spec := ""
			m := Model{
				Store:        st,
				HarnessReady: true,
				MuxConnected: true,
				InstallReady: true,
				InstallSpec:  &spec,
				OnRefresh:    func() {},
			}
			tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 800)
			tt.Frame()
			assertMarketCursors(t, "market-"+locale, tt)
		})
	}
}
