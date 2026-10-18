package main

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
)

func TestUnifiedShortcutNativeProductPersistence(t *testing.T) {
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = cfg.SavePageShortcuts(map[string]string{"session.search": "Mod+Alt+F11"}); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg}
	prefs := s.pageShortcutPreferences()
	snap := prefs.Get()
	if snap.Status != "ready" {
		t.Fatalf("init %#v", snap)
	}
	if s.pageShortcutBindings()["session.search"] != "Mod+Alt+F11" {
		t.Fatal("legacy override not consumed")
	}
	// Product catalog and native catalog share the same command ID/default.
	profile := "desktop:" + shortcutPlatform()
	snap = prefs.SetDefinitions("product", []shortcutprefs.Definition{{ID: "session.search", Defaults: map[string]shortcutprefs.Binding{profile: {Code: "KeyK", Modifiers: []string{"primary"}}}}})
	e := s.pageShortcutEditor()
	*e.Draft("session.search") = "Mod+Alt+F12"
	product := prefs.Apply("product", snap.Revision, []shortcutprefs.Edit{{Type: "set", ID: "session.search", Binding: &shortcutprefs.Binding{Code: "KeyJ", Modifiers: []string{"primary", "alt"}}}})
	if product.Status != "saved" {
		t.Fatalf("product %#v", product)
	}
	if e.Save() {
		t.Fatal("native stale draft overwrote product")
	}
	e.Discard()
	e = s.pageShortcutEditor()
	if *e.Draft("session.search") != "Mod+Alt+J" {
		t.Fatalf("product edit not consumed %s", *e.Draft("session.search"))
	}
	*e.Draft("session.search") = "Mod+Alt+F12"
	if !e.Save() {
		t.Fatal(e.Err)
	}
	if got := prefs.Effective("product")[0].Binding; got.Code != "F12" {
		t.Fatalf("native edit not visible to product %#v", got)
	}
	// A normal settings update preserves the authoritative raw shortcut document.
	before, _ := cfg.ShortcutDocument()
	if err = cfg.Update(func(c *config.Config) { c.Theme = "deepseek" }); err != nil {
		t.Fatal(err)
	}
	after, _ := cfg.ShortcutDocument()
	if string(before) != string(after) {
		t.Fatal("ordinary config update clobbered preferences")
	}
	e.ResetDraft("session.search")
	if !e.Save() {
		t.Fatal(e.Err)
	}
	raw, _ := cfg.ShortcutDocument()
	var doc shortcutprefs.Document
	if json.Unmarshal(raw, &doc) != nil {
		t.Fatal("document")
	}
	if _, ok := doc.Profiles[profile][shortcuts.CanonicalID("session.search")]; ok {
		t.Fatal("reset stored default override")
	}
	// A fresh mainState must not resurrect the preserved legacy migration source.
	fresh := (&mainState{cfgStore: cfg}).pageShortcutPreferences()
	if _, ok := fresh.Get().Document.Profiles[profile]["session.search"]; ok {
		t.Fatal("legacy override resurrected")
	}
}
