package settings

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
)

func TestDSK10NativeEntry(t *testing.T) {
	if _, ok := reflect.TypeOf(InterfaceAccess{}).FieldByName("Shortcuts"); !ok {
		t.Fatal("Interface settings has no native shortcut preferences entry")
	}
}

func TestDSK10ShortcutsPaneInteractions(t *testing.T) {
	writes := 0
	patches := []map[string]string{}
	e := shortcuts.NewEditor(nil, nil, func(patch map[string]string) error {
		writes++
		copy := map[string]string{}
		for id, value := range patch {
			copy[id] = value
		}
		patches = append(patches, copy)
		return nil
	})
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
	// The old fixture chord is now the real session.fork default. Ensure
	// conflict validation still blocks save, then use an unoccupied chord.
	tt.Type("Mod+Shift+J")
	if err := tt.Click("save-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 0 || e.Active()["settings.open"] != "Mod+," {
		t.Fatal("conflicting draft was saved")
	}
	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Mod+Alt+F12")
	if err := tt.Click("save-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 1 || e.Active()["settings.open"] != "Mod+Alt+F12" {
		t.Fatalf("native edit/save: %d %v", writes, e.Active())
	}

	// Fixed original Editor.tsx submits Clear and Reset immediately for only
	// the current command. An unrelated manual draft must remain unsubmitted.
	*e.Draft("session.search") = "Mod+Alt+F11"
	if err := tt.Click("unbind-settings.open"); err != nil {
		t.Fatal(err)
	}
	if writes != 2 || e.Active()["settings.open"] != "" || *e.Draft("settings.open") != "" {
		t.Fatal("Unbind did not persist immediately", writes, e.Active(), e.Err)
	}
	if !reflect.DeepEqual(patches[1], map[string]string{"settings.open": ""}) {
		t.Fatal("Unbind submitted more than its own command", patches[1])
	}
	if *e.Draft("session.search") != "Mod+Alt+F11" || e.Active()["session.search"] == "Mod+Alt+F11" {
		t.Fatal("Unbind committed or discarded an unrelated draft")
	}
	if err := tt.Click("default-settings.open"); err != nil {
		t.Fatal(err)
	}
	if writes != 3 || *e.Draft("settings.open") != "Mod+," || e.Active()["settings.open"] != "Mod+," {
		t.Fatal("single default did not persist immediately", writes, e.Active(), e.Err)
	}
	if !reflect.DeepEqual(patches[2], map[string]string{"settings.open": "Mod+,"}) {
		t.Fatal("single default submitted more than its own command", patches[2])
	}
	if *e.Draft("session.search") != "Mod+Alt+F11" || e.Active()["session.search"] == "Mod+Alt+F11" {
		t.Fatal("single default committed or discarded an unrelated draft")
	}
	if err := tt.Click("reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 3 {
		t.Fatal("reset wrote before confirmation")
	}
	if err := tt.Click("cancel-reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 3 || e.ResetPending || *e.Draft("session.search") != "Mod+Alt+F11" {
		t.Fatal("reset cancel wrote or discarded draft")
	}
	if err := tt.Click("reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("confirm-reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 4 || e.Active()["settings.open"] != "Mod+," || e.Active()["session.search"] != shortcuts.Defaults()["session.search"] || *e.Draft("session.search") != shortcuts.Defaults()["session.search"] {
		t.Fatal("confirmed reset did not restore and persist known defaults", writes, e.Active(), e.Err)
	}
}

func TestShortcutImmediateRowSaveFailureRetainsAcceptedBindingAndDrafts(t *testing.T) {
	for _, operation := range []struct{ button, value string }{
		{"unbind-settings.open", ""}, {"default-settings.open", "Mod+,"},
	} {
		t.Run(operation.button, func(t *testing.T) {
			writes := 0
			var submitted map[string]string
			e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12"}, nil, func(patch map[string]string) error {
				writes++
				submitted = patch
				return errors.New("synthetic persistence failure")
			})
			e.Query = "settings"
			*e.Draft("session.search") = "Mod+Alt+F11"
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { shortcutsPane(c, e, ui.RGB(20, 20, 20), ui.RGB(80, 80, 80)) })
			}, 1000, 1200)
			if err := tt.Click(operation.button); err != nil {
				t.Fatal(err)
			}
			if writes != 1 || !reflect.DeepEqual(submitted, map[string]string{"settings.open": operation.value}) {
				t.Fatal("row gesture did not make one narrow persistence attempt", writes, submitted)
			}
			if e.Active()["settings.open"] != "Mod+Alt+F12" || *e.Draft("settings.open") != operation.value || !strings.Contains(e.Err, "synthetic persistence failure") {
				t.Fatal("failed immediate write changed accepted binding or lost candidate", e.Active(), e.Values(), e.Err)
			}
			if *e.Draft("session.search") != "Mod+Alt+F11" || e.Active()["session.search"] == "Mod+Alt+F11" {
				t.Fatal("failed row write affected another command draft")
			}
			tt.Frame()
			if writes != 1 {
				t.Fatal("failed immediate write retried without a user gesture")
			}
		})
	}
}
