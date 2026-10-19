package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
)

func TestShortcutReloadButtonInvokesExplicitReaderAndClearsDraft(t *testing.T) {
	editor := shortcuts.NewEditor(nil, nil, nil)
	editor.Query = "settings"
	*editor.Draft("settings.open") = "Mod+Alt+F12"
	reads := 0
	editor.SetReload(func() (map[string]string, error) {
		reads++
		values := editor.Active()
		values["settings.open"] = "Mod+Alt+F11"
		return values, nil
	})
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { shortcutsPane(c, editor, ui.RGB(20, 20, 20), ui.RGB(80, 80, 80)) })
	}, 1000, 1200)
	if err := tester.Click("reload-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if reads != 1 || editor.Dirty() || *editor.Draft("settings.open") != "Mod+Alt+F11" {
		t.Fatal("reload button only discarded memory", reads, editor.Values())
	}
}
