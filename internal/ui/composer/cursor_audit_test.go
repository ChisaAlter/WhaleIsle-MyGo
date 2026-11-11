package composer

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// TestCursorAuditComposer renders the composer headlessly and asserts every
// enabled clickable control uses the pointing hand cursor.
func TestCursorAuditComposer(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	m := permissionModelFixture()
	m.OnPermissionChange = func(string) {}
	m.OnModelChange = func(string) {}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 880, 420)
	tt.Frame()
	boxes := tt.Interactives()
	var lines []string
	for _, b := range boxes {
		if b.Cursor == ui.CursorPointer {
			continue
		}
		if slices.Contains(b.Kinds, "click") && !slices.Contains(b.Kinds, "text") {
			t.Errorf("clickable composer element missing CursorPointer: %s", b.String())
		}
		lines = append(lines, b.String())
	}
	sort.Strings(lines)
	t.Logf("%d interactive elements, %d without a hand:\n%s\ntexts: %q", len(boxes), len(lines), strings.Join(lines, "\n"), tt.Texts())
}
