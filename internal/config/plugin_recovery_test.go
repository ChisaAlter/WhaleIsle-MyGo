package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdatePluginRecoveryAndClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"port":4000,"pluginRecovery":{"skipUserPlugins":false,"reason":"manual"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdatePluginRecovery(func(m map[string]any) {
		m["skipUserPlugins"] = true
	}); err != nil {
		t.Fatal(err)
	}
	if c := s.Get(); !c.SkipUserPlugins {
		t.Fatal("SkipUserPlugins not projected")
	}
	rec := s.PluginRecovery()
	if !rec.Present || !rec.SkipUserPlugins || rec.Reason != "manual" {
		t.Fatalf("recovery = %+v", rec)
	}
	if err := s.ClearPluginRecovery(); err != nil {
		t.Fatal(err)
	}
	if c := s.Get(); c.SkipUserPlugins {
		t.Fatal("SkipUserPlugins still set")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["pluginRecovery"]; ok {
		t.Fatalf("pluginRecovery still present: %v", raw["pluginRecovery"])
	}
}
