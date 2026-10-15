package settings

import (
	"github.com/egoist/mygo/ui"
	"reflect"
	"testing"
	"whaleisle/internal/shortcuts"
)

func TestDSK10NativeEntry(t *testing.T) {
	if _, ok := reflect.TypeOf(InterfaceAccess{}).FieldByName("Shortcuts"); !ok {
		t.Fatal("Interface settings has no native shortcut preferences entry")
	}
}

func TestDSK10ShortcutsPaneInteractions(t *testing.T) {
	writes := 0
	e := shortcuts.NewEditor(nil, nil, func(map[string]string) error { writes++; return nil })
	e.Query = "settings"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { shortcutsPane(c, e, ui.RGB(20, 20, 20), ui.RGB(80, 80, 80)) })
	}, 1000, 1200)
	if _, ok := tt.Find("shortcut-terminal.toggle"); ok {
		t.Fatal("search did not filter rows")
	}
	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Mod+Shift+J")
	if err := tt.Click("save-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 1 || e.Active()["settings.open"] != "Mod+Shift+J" {
		t.Fatalf("native edit/save: %d %v", writes, e.Active())
	}
	if err := tt.Click("unbind-settings.open"); err != nil {
		t.Fatal(err)
	}
	if e.Active()["settings.open"] == "" {
		t.Fatal("unbind draft applied before save")
	}
	if err := tt.Click("save-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if e.Active()["settings.open"] != "" {
		t.Fatal("unbind not saved")
	}
	if err := tt.Click("default-settings.open"); err != nil {
		t.Fatal(err)
	}
	if *e.Draft("settings.open") != "Mod+," || e.Active()["settings.open"] != "" {
		t.Fatal("single default not a draft")
	}
	if err := tt.Click("reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 2 {
		t.Fatal("reset wrote before confirmation")
	}
	if err := tt.Click("cancel-reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 2 || e.ResetPending {
		t.Fatal("reset cancel failed")
	}
	if err := tt.Click("reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("confirm-reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 3 || e.Active()["settings.open"] != "Mod+," {
		t.Fatal("confirmed reset failed")
	}
}
