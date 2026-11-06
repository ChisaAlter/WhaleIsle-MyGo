package config

import (
	"errors"
	"testing"
	"time"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

func TestNamespaceSavesRespectHeldCommitLock(t *testing.T) {
	for name, save := range map[string]func(*Store) error{
		"appearance":     func(s *Store) error { return s.SaveAppearance(theme.Defaults()) },
		"desktop pet":    func(s *Store) error { return s.SaveDesktopPet(DesktopPet{Enabled: true}) },
		"page shortcuts": func(s *Store) error { return s.SavePageShortcuts(map[string]string{"action": "Ctrl+K"}) },
		"pet care":       func(s *Store) error { return s.PatPet(time.UnixMilli(1_790_000_000_000)) },
		"plugin recovery": func(s *Store) error {
			return s.UpdatePluginRecovery(func(value map[string]any) { value["skipUserPlugins"] = true })
		},
		"shortcut document": func(s *Store) error { return s.SaveShortcutDocument([]byte(`{"version":1}`)) },
		"wallpaper":         func(s *Store) error { return s.SaveWallpaperBackground(wallpapers.Defaults()) },
		"workspace save": func(s *Store) error {
			return s.SaveWorkspaceNavigationPreferences("host", WorkspaceNavigationPreferences{GroupBy: "flat"})
		},
		"workspace merge": func(s *Store) error {
			_, err := s.MergeWorkspaceNavigationPreferences("host", WorkspaceNavigationPreferences{}, WorkspaceNavigationPreferences{GroupBy: "flat"})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			const initial = `{"port":4000,"future":{"keep":true}}`
			writeConfig(t, dir, initial)
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			release, err := acquireCommitLock(s.path)
			if err != nil {
				t.Fatal(err)
			}
			err = save(s)
			release()
			if err == nil || errors.Is(err, ErrConflict) {
				t.Fatalf("want lock-busy error, got %v", err)
			}
			if got := readConfig(t, dir); got != initial {
				t.Fatalf("save bypassed held lock: %s", got)
			}
			if err := save(s); err != nil {
				t.Fatalf("save after release: %v", err)
			}
			if got := readConfig(t, dir); got == initial {
				t.Fatal("save did not persist after release")
			}
		})
	}
}
