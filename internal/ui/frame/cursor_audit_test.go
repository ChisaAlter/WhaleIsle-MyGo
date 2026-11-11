package frame

import (
	"sort"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestCursorAuditHome renders the whole frame headlessly and reports every
// enabled interactive element whose cursor is not the pointing hand, so a
// missing hand is found by the test instead of by a person using the app.
func TestCursorAuditHome(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)
	store := session.New()
	layout := DefaultLayout()
	draft := ""
	tt := ui.NewTester(func(c *ui.Context) {
		pass := store.BeginFrame()
		defer pass.EndFrame()
		m := Model{Store: store, Sizes: &layout, Draft: &draft, ThemeID: "deepseek.light", FrameView: pass}
		c.SetTheme(ui.LightTheme())
		ui.Column(c).Fill().Children(func() { View(c, m) })
	}, 1440, 864)
	tt.Frame()
	boxes := tt.Interactives()
	var lines []string
	for _, b := range boxes {
		if b.Cursor == ui.CursorPointer {
			continue
		}
		lines = append(lines, b.String())
	}
	sort.Strings(lines)
	t.Logf("%d interactive elements, %d without a hand:\n%s", len(boxes), len(lines), strings.Join(lines, "\n"))
}
