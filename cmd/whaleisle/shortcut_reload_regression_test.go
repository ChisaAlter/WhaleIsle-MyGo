package main

import (
	"encoding/json"
	"os"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/shortcutprefs"
)

func TestShortcutExplicitReloadDiscardsDraftAndPreservesExternalConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg}
	editor := s.pageShortcutEditor()
	*editor.Draft("session.search") = "Mod+Alt+F12"
	external, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := external.Update(func(c *config.Config) { c.Workspace = "external-workspace"; c.Locale = "en" }); err != nil {
		t.Fatal(err)
	}
	profile := "desktop:" + shortcutPlatform()
	doc := shortcutprefs.Document{SchemaVersion: 2, Profiles: map[string]map[string]*shortcutprefs.Binding{profile: {
		"session.search": {Code: "KeyJ", Modifiers: []string{"primary", "alt"}},
	}}}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := external.SaveShortcutDocument(raw); err != nil {
		t.Fatal(err)
	}
	if editor.Save() {
		t.Fatal("dirty stale editor overwrote external preferences")
	}
	if !editor.Reload() {
		t.Fatal(editor.ReadError)
	}
	if editor.ReadError != "" || editor.Dirty() || *editor.Draft("session.search") != "Mod+Alt+J" {
		t.Fatal("explicit reload did not replace discarded draft", editor.ReadError, editor.Values())
	}
	*editor.Draft("session.search") = "Mod+Alt+F11"
	if !editor.Save() {
		t.Fatal(editor.Err)
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Get(); got.Workspace != "external-workspace" || got.Locale != "en" {
		t.Fatal("narrow shortcut save resurrected unrelated old values", got.Workspace, got.Locale)
	}
	// Reload is not permission to overwrite a later external edit.
	if err := reopened.Update(func(c *config.Config) { c.Locale = "zh" }); err != nil {
		t.Fatal(err)
	}
	*editor.Draft("session.search") = "Mod+Alt+F12"
	if editor.Save() {
		t.Fatal("later external edit bypassed CAS after reload")
	}
}

func TestShortcutReloadRecoversInitializationErrorAndMalformedRepair(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	external, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := external.SavePageShortcuts(map[string]string{"session.search": "Mod+Alt+F11"}); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg}
	editor := s.pageShortcutEditor()
	if editor.ReadError == "" {
		t.Fatal("stale initialization unexpectedly accepted")
	}
	if !editor.Reload() || editor.ReadError != "" || *editor.Draft("session.search") != "Mod+Alt+F11" {
		t.Fatal("initialization error reader remained pinned", editor.ReadError)
	}
	valid, err := os.ReadFile(cfg.Path())
	if err != nil {
		t.Fatal(err)
	}
	malformed := []byte(`{"goUnifiedShortcuts":`)
	if err := os.WriteFile(cfg.Path(), malformed, 0600); err != nil {
		t.Fatal(err)
	}
	if editor.Reload() || editor.ReadError == "" {
		t.Fatal("malformed reload reported success")
	}
	after, err := os.ReadFile(cfg.Path())
	if err != nil || string(after) != string(malformed) {
		t.Fatal("failed reload changed source", err)
	}
	if err := os.WriteFile(cfg.Path(), valid, 0600); err != nil {
		t.Fatal(err)
	}
	if !editor.Reload() || editor.ReadError != "" {
		t.Fatal("repaired source remained unreadable", editor.ReadError)
	}
}
