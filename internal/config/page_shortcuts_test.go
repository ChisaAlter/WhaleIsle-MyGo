package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDSK10ShortcutPersistencePreservesUnknownAndFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	original := []byte(`{"unknown":{"token":"fixture"},"goPageShortcuts":{"future":{"nested":true},"settings.open":"Mod+,"}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.rename = func(string, string) error { return errors.New("injected rename failure") }
	if err = s.SavePageShortcuts(map[string]string{"settings.open": "Mod+Shift+J"}); err == nil {
		t.Fatal("failure hidden")
	}
	disk, _ := os.ReadFile(path)
	if string(disk) != string(original) {
		t.Fatal("failed save changed original")
	}
	got, _ := s.PageShortcuts([]string{"settings.open"})
	if got["settings.open"] != "Mod+," {
		t.Fatal("failed save changed active config")
	}
	s.rename = nil
	if err = s.SavePageShortcuts(map[string]string{"settings.open": "Mod+Shift+J", "tools.toggle": ""}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err = reopened.PageShortcuts([]string{"settings.open", "tools.toggle"})
	if err != nil || got["settings.open"] != "Mod+Shift+J" || got["tools.toggle"] != "" {
		t.Fatalf("reopen %v %v", got, err)
	}
	disk, _ = os.ReadFile(path)
	var root map[string]json.RawMessage
	_ = json.Unmarshal(disk, &root)
	var nested map[string]json.RawMessage
	_ = json.Unmarshal(root["goPageShortcuts"], &nested)
	if string(root["unknown"]) == "" || string(nested["future"]) == "" {
		t.Fatal("unknown fields lost")
	}
	if err = reopened.Update(func(c *Config) { c.Theme = "deepseek" }); err != nil {
		t.Fatal(err)
	}
	got, _ = reopened.PageShortcuts([]string{"settings.open"})
	if got["settings.open"] != "Mod+Shift+J" {
		t.Fatal("ordinary config update lost shortcuts")
	}
	if err = os.WriteFile(path, []byte(`{"external":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(reopened.SavePageShortcuts(map[string]string{"settings.open": "Mod+,"}), ErrConflict) {
		t.Fatal("external edit overwritten")
	}
}

func TestDSK10ShortcutMalformedKnownFieldDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"goPageShortcuts":{"settings.open":123,"future":{"ok":true}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.PageShortcuts([]string{"settings.open"}); err == nil {
		t.Fatal("malformed known field accepted")
	}
	if _, err = s.PageShortcuts([]string{"missing"}); err != nil {
		t.Fatalf("unknown fields must stay opaque: %v", err)
	}
}
