package shortcuts

import "testing"

func TestExtendedFunctionKeysAndRecording(t *testing.T) {
	for _, key := range []string{"Mod+F1", "Mod+F5", "Mod+Shift+F12"} {
		if _, err := Parse(key); err != nil {
			t.Fatal(key, err)
		}
	}
	for _, key := range []string{"F5", "Alt+F4", "Mod+F13", "Mod+S"} {
		if _, err := parseForPlatform(key, "linux"); err == nil {
			t.Fatal("accepted", key)
		}
	}
	writes := 0
	e := NewEditor(nil, nil, func(map[string]string) error { writes++; return nil })
	e.BeginRecording("window.refresh")
	if !e.Record("Mod+Shift+F7") {
		t.Fatal(e.Err)
	}
	if *e.Draft("window.refresh") != "Mod+Shift+F7" || e.Active()["window.refresh"] != "Mod+F5" || writes != 0 {
		t.Fatal("recording must change draft only")
	}
	e.BeginRecording("window.refresh")
	if e.Record("Mod+UnknownKey") {
		t.Fatal("invalid key accepted")
	}
	e.CancelRecording()
	if !e.Save() || e.Active()["window.refresh"] != "Mod+Shift+F7" || writes != 1 {
		t.Fatal(e.Err)
	}
}
