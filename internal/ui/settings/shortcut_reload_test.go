package settings

import (
	"testing"

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
	tester := shortcutDialogFixture(t, editor, 900, 800)
	if err := tester.Click("reload-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if reads != 1 || editor.Dirty() || *editor.Draft("settings.open") != "Mod+Alt+F11" {
		t.Fatal("reload button only discarded memory", reads, editor.Values())
	}
}
