package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspacePreferencesPersistenceFixture(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, FileName)
	initial := []byte(`{"unrelated":{"preserve":true},"goWorkspaceNavigation":{"host-a":{"future":7}}}`)
	if err := os.WriteFile(file, initial, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := WorkspaceNavigationPreferences{GroupBy: "workspace-tree", OrderBy: "updated", Collapsed: map[string]bool{"id-and-path-a": true}}
	b := WorkspaceNavigationPreferences{GroupBy: "flat", OrderBy: "manual", Collapsed: map[string]bool{"id-and-path-b": false}}
	if err = s.SaveWorkspaceNavigationPreferences("host-a", a); err != nil {
		t.Fatal(err)
	}
	a.Collapsed["id-and-path-a"] = false
	if err = s.SaveWorkspaceNavigationPreferences("host-b", b); err != nil {
		t.Fatal(err)
	}
	if err = s.Update(func(c *Config) { c.Theme = "deepseek" }); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.WorkspaceNavigationPreferences("host-a")
	if err != nil || got.GroupBy != "workspace-tree" || !got.Collapsed["id-and-path-a"] {
		t.Fatal("roundtrip or input alias", got, err)
	}
	got.Collapsed["id-and-path-a"] = false
	again, _ := reopened.WorkspaceNavigationPreferences("host-a")
	if !again.Collapsed["id-and-path-a"] {
		t.Fatal("read aliases config")
	}
	bg, _ := reopened.WorkspaceNavigationPreferences("host-b")
	if bg.GroupBy != "flat" || bg.Collapsed["id-and-path-a"] {
		t.Fatal("Host preferences crossed")
	}
	raw, _ := os.ReadFile(file)
	var all map[string]any
	_ = json.Unmarshal(raw, &all)
	if all["unrelated"].(map[string]any)["preserve"] != true || all[workspaceNavigationKey].(map[string]any)["host-a"].(map[string]any)["future"] != float64(7) {
		t.Fatal("unknown config fields erased")
	}
	reopened.rename = func(string, string) error { return errors.New("fixture disk write failure") }
	changed := WorkspaceNavigationPreferences{GroupBy: "flat", Collapsed: map[string]bool{}}
	if err = reopened.SaveWorkspaceNavigationPreferences("host-a", changed); err == nil {
		t.Fatal("write failure ignored")
	}
	after, _ := os.ReadFile(file)
	if string(after) != string(raw) {
		t.Fatal("failed write changed disk")
	}
	unchanged, _ := reopened.WorkspaceNavigationPreferences("host-a")
	if unchanged.GroupBy != "workspace-tree" {
		t.Fatal("failed write changed committed config")
	}
	reopened.rename = nil
	if err = os.WriteFile(file, []byte(`{"external":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = reopened.SaveWorkspaceNavigationPreferences("host-a", changed); !errors.Is(err, ErrConflict) {
		t.Fatal("external edit was overwritten", err)
	}
}
