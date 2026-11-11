package pluginmanager

import (
	"context"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func assertPluginManagerCursors(t *testing.T, viewName string, tt *ui.Tester) {
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

func TestPluginManagerParityAndCursorAudit(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	m := pageModel(t)
	owner := m.Controller.Snapshot().Owner
	if err := m.Controller.LoadIndexes(context.Background(), owner); err != nil {
		t.Fatal(err)
	}

	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
	tt.Frame()
	assertPluginManagerCursors(t, "installed-list", tt)

	if err := tt.Click("Fixture plugin"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	assertPluginManagerCursors(t, "package-detail", tt)

	if err := tt.Click("Back to plugins"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	if err := tt.Click("Discover"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	assertPluginManagerCursors(t, "discover-list", tt)

	if err := tt.Click("Index sources"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	assertPluginManagerCursors(t, "index-sources", tt)
}
