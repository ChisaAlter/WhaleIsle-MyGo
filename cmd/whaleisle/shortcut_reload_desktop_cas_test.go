package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/paths"
	"whaleisle/internal/shortcutprefs"
)

// Exercise the actual desktop-form save callback after shortcut-only recovery.
// Every persistence target is a temporary directory. The isolated integration
// guard also suppresses login registration if a regression reaches effects.
func TestShortcutReloadDoesNotAuthorizeStaleDesktopDraft(t *testing.T) {
	for _, saveShortcut := range []bool{false, true} {
		name := "reload-only"
		if saveShortcut {
			name = "reload-and-shortcut-save"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("WHALEISLE_ISOLATED", "1")
			t.Setenv(paths.EnvUserData, dir)
			cfg, err := config.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.Update(func(c *config.Config) {
				c.Workspace = "fixture-before"
				c.Locale = "zh"
				c.WallpaperPath = "" // Never read or import a real wallpaper.
			}); err != nil {
				t.Fatal(err)
			}
			s := &mainState{cfgStore: cfg}
			s.seedDesktopDraft()
			editor := s.pageShortcutEditor()
			*editor.Draft("session.search") = "Mod+Alt+F12"
			external, err := config.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err := external.Update(func(c *config.Config) {
				c.Workspace = "fixture-external"
				c.Locale = "en"
			}); err != nil {
				t.Fatal(err)
			}
			profile := "desktop:" + shortcutPlatform()
			raw, err := json.Marshal(shortcutprefs.Document{SchemaVersion: 2,
				Profiles: map[string]map[string]*shortcutprefs.Binding{profile: {
					"session.search": {Code: "KeyJ", Modifiers: []string{"primary", "alt"}},
				}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := external.SaveShortcutDocument(raw); err != nil {
				t.Fatal(err)
			}
			if !editor.Reload() || editor.Dirty() || *editor.Draft("session.search") != "Mod+Alt+J" {
				t.Fatal("shortcut-only reload did not adopt repaired preferences", editor.ReadError)
			}
			if saveShortcut {
				*editor.Draft("session.search") = "Mod+Alt+F11"
				if !editor.Save() {
					t.Fatal("shortcut snapshot could not save independently", editor.Err)
				}
			}
			// A shortcut discard must not also discard the separate desktop form.
			if s.desktopDraft.workspace != "fixture-before" || s.desktopDraft.locale != "zh" {
				t.Fatal("shortcut reload silently replaced another form's draft")
			}
			before, err := os.ReadFile(cfg.Path())
			if err != nil {
				t.Fatal(err)
			}
			s.desktopDraft.closeToTray = !s.desktopDraft.closeToTray
			s.saveDesktopConfig()
			if !strings.Contains(s.desktopSaveErr, config.ErrConflict.Error()) || s.desktopLastAction != "" {
				t.Fatal("stale desktop-form save was not rejected by CAS", s.desktopSaveErr, s.desktopLastAction)
			}
			after, err := os.ReadFile(cfg.Path())
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("stale desktop-form save changed disk bytes", err)
			}
			reopened, err := config.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if got := reopened.Get(); got.Workspace != "fixture-external" || got.Locale != "en" {
				t.Fatal("desktop callback resurrected pre-reload fields", got.Workspace, got.Locale)
			}
		})
	}
}
