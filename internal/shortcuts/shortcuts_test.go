package shortcuts

import (
	"errors"
	"testing"
)

func TestDSK10RegistryValidationAndSearch(t *testing.T) {
	if len(Commands()) != 7 {
		t.Fatal("expected seven bindings, including tools alternate")
	}
	for _, raw := range []string{"B", "Alt+F4", "Win+B", "Mod+Space", "Mod+S", "Mod+Q", "Mod+Shift+V", "Mod+Mod+B", "Mod++B", "Mod+Banana"} {
		if _, err := Parse(raw); err == nil {
			t.Errorf("unsafe/invalid binding accepted: %s", raw)
		}
	}
	chord, err := Parse(" control + shift + j ")
	if err != nil || chord.String() != "Mod+Shift+J" {
		t.Fatalf("parse: %+v %v", chord, err)
	}
	if len(Validate(map[string]string{"settings.open": "Mod+B"})) != 2 {
		t.Fatal("both conflicting rows must be marked")
	}
	e := NewEditor(nil, nil, func(map[string]string) error { return nil })
	if len(e.Search("terminal")) != 1 || len(e.Search("终端")) != 1 || len(e.Search("Mod+Alt+B")) != 1 {
		t.Fatal("search did not include label or binding")
	}
}

func TestDSK10EditorDraftCommitFailureAndReset(t *testing.T) {
	fail := true
	writes := 0
	e := NewEditor(nil, nil, func(v map[string]string) error {
		writes++
		if fail {
			return errors.New("disk full")
		}
		if v["settings.open"] == "" {
			t.Fatal("save dropped a binding")
		}
		return nil
	})
	*e.Draft("settings.open") = "Mod+Shift+J"
	if e.Save() || e.Active()["settings.open"] != "Mod+," || *e.Draft("settings.open") != "Mod+Shift+J" {
		t.Fatal("failed save must retain draft and previous active binding")
	}
	fail = false
	if !e.Save() || e.Active()["settings.open"] != "Mod+Shift+J" {
		t.Fatal("rebind not published after persistence")
	}
	*e.Draft("tools.toggle") = ""
	if !e.Save() || e.Active()["tools.toggle"] != "" {
		t.Fatal("explicit unbind not committed")
	}
	*e.Draft("settings.open") = "Mod+B"
	if e.Save() || writes != 3 {
		t.Fatal("conflict must not write")
	}
	e.RequestReset()
	e.CancelReset()
	if e.ConfirmReset() || writes != 3 {
		t.Fatal("cancelled reset wrote")
	}
	e.RequestReset()
	if !e.ConfirmReset() || e.Active()["settings.open"] != "Mod+," || e.Active()["tools.toggle"] != "Mod+\\" {
		t.Fatal("confirmed reset not committed")
	}
}

func TestDSK10InvalidSavedBindingsFailClosed(t *testing.T) {
	e := NewEditor(map[string]string{"settings.open": "Mod+B", "terminal.toggle": "Alt+F4"}, nil, nil)
	live := Resolve(e.Active())
	if _, ok := live["settings.open"]; ok {
		t.Fatal("conflicting settings key remains active")
	}
	if _, ok := live["sidebar.toggle"]; ok {
		t.Fatal("conflicting sidebar remains active")
	}
	if _, ok := live["terminal.toggle"]; ok {
		t.Fatal("unsafe saved key remains active")
	}
	e = NewEditor(nil, errors.New("malformed preferences"), nil)
	if len(Resolve(e.Active())) != 0 || e.Save() {
		t.Fatal("unreadable configuration must not silently use defaults or save")
	}
}

func TestDSK10ReservedSystemKeys(t *testing.T) {
	for _, raw := range []string{"Mod+`", "Mod+Shift+`", "Mod+Shift+3", "Mod+Shift+4", "Mod+Shift+5", "Mod+Shift+6", "Mod+Alt+D"} {
		if _, err := parseForPlatform(raw, "darwin"); err == nil {
			t.Errorf("macOS reserved key accepted: %s", raw)
		}
	}
	if _, err := parseForPlatform("Mod+`", "windows"); err != nil {
		t.Fatal("Windows terminal default changed")
	}
	for _, command := range commandsForPlatform("darwin") {
		if command.ID == "terminal.toggle" && command.Default != "" {
			t.Fatal("macOS default captures window switching")
		}
	}
}
