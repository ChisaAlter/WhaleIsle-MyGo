package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProjectPreferenceDefaultsAndSingleFieldRoundTrip(t *testing.T) {
	if Defaults().ProjectEnabled {
		t.Fatal("installed Project defaults off")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	original := []byte(`{"projectEnabled":true,"workspace":"owned","custom":{"projectData":"keep"},"pluginRecovery":{"skipUserPlugins":false}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := store.Get()
	if !before.ProjectEnabled {
		t.Fatal("existing accepted Project key was ignored")
	}
	if err = store.Update(func(c *Config) { c.ProjectEnabled = false }); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := before
	want.ProjectEnabled = false
	if !reflect.DeepEqual(reopened.Get(), want) {
		t.Fatal("Project toggle changed unrelated typed config")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]json.RawMessage
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	var custom struct {
		ProjectData string `json:"projectData"`
	}
	var recovery struct {
		SkipUserPlugins *bool `json:"skipUserPlugins"`
	}
	if json.Unmarshal(value["custom"], &custom) != nil || custom.ProjectData != "keep" || json.Unmarshal(value["pluginRecovery"], &recovery) != nil || recovery.SkipUserPlugins == nil || *recovery.SkipUserPlugins {
		t.Fatalf("Project toggle removed unknown state:%s", raw)
	}
}
