package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

func TestNamespaceSubObjectUnknownFieldPreservation(t *testing.T) {
	dir := t.TempDir()
	initial := map[string]any{
		"port": 4000,
		"goPluginInstallRegistry": map[string]any{
			"kind":         "custom",
			"registry":     "https://registry.example.test",
			"futureNested": map[string]any{"mirror": "fast"},
		},
		"goWallpaperBackground": map[string]any{
			"glassOpacity":    80,
			"futureBgTopKey":  "keep-bg",
			"wallpaperCrop":   map[string]any{"aspect": 1.5, "zoom": 1, "x": 0.5, "y": 0.5, "cropExtra": 9},
			"wallpaperSources": []any{map[string]any{"id": "src-1", "kind": "custom", "name": "Old", "sourceExtra": true}},
			"wallpaperFavorites": []any{
				map[string]any{"id": "fav-1", "sourceId": "src-1", "title": "T1", "thumbUrl": "t", "imageUrl": "i", "favExtra": "keep-fav"},
			},
		},
		"goAppearance": map[string]any{
			"light":               "deepseek",
			"dark":                "deepseek",
			"futureAppearanceKey": "keep-app",
			"interface":           map[string]any{"family": "Inter", "size": 16, "fontHint": "subpixel"},
			"chat":                map[string]any{"family": "Inter", "size": 14, "lineHeight": 1.4},
			"custom": []any{
				map[string]any{
					"id":          "theme-1",
					"name":        "Old Name",
					"familyExtra": "keep-fam",
					"light":       map[string]any{"background": "#fff", "foreground": "#000", "accent": "#111", "seedExtra": "light-keep"},
					"dark":        map[string]any{"background": "#000", "foreground": "#fff", "accent": "#222", "seedExtra": "dark-keep"},
				},
			},
		},
		"mygoDesktopPet": map[string]any{
			"enabled":       true,
			"skinId":        "miku",
			"neural":        true,
			"futurePetFlag": "keep-pet",
			"behavior":      map[string]any{"scale": 1, "opacity": 1, "personality": "natural", "activity": "balanced", "behaviorExtra": 7},
			"position":      map[string]any{"x": 10, "y": 20, "posExtra": "drag-v2"},
		},
	}
	rawBytes, err := json.MarshalIndent(initial, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeConfig(t, dir, string(rawBytes))

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Update typed Config without losing nested key in goPluginInstallRegistry.
	if err := s.Update(func(c *Config) { c.Port = 4100 }); err != nil {
		t.Fatal(err)
	}

	// 2. SaveWallpaperBackground preserves unknown keys at bg top, crop, source, favorite.
	wp, err := s.WallpaperBackground()
	if err != nil {
		t.Fatal(err)
	}
	wp.Glass = 65
	wp.Crop.Zoom = 2
	wp.Sources = []wallpapers.Source{{ID: "src-1", Kind: "custom", Name: "Updated"}}
	wp.Favorites = []wallpapers.Item{{ID: "fav-1", SourceID: "src-1", Title: "T2", ThumbURL: "t", ImageURL: "i"}}
	if err := s.SaveWallpaperBackground(wp); err != nil {
		t.Fatal(err)
	}

	// 3. SaveAppearance preserves unknown keys in goAppearance, fonts, custom family & seeds.
	app, err := s.Appearance()
	if err != nil {
		t.Fatal(err)
	}
	app.Interface.Size = 18
	app.Chat.Size = 15
	app.Custom = []theme.Family{{
		ID:    "theme-1",
		Name:  "New Name",
		Light: theme.Seed{Background: "#eeeeee", Foreground: "#111111", Accent: "#333333"},
		Dark:  theme.Seed{Background: "#111111", Foreground: "#eeeeee", Accent: "#444444"},
	}}
	if err := s.SaveAppearance(app); err != nil {
		t.Fatal(err)
	}

	// 4. SaveDesktopPet preserves unknown keys in mygoDesktopPet, behavior, and position.
	pet := s.DesktopPet()
	pet.Behavior.Scale = 1.4
	pet.Position = &PetPosition{X: 100, Y: 200, Space: PetPositionWindowsPhysical}
	if err := s.SaveDesktopPet(pet); err != nil {
		t.Fatal(err)
	}

	var onDisk map[string]any
	if err := json.Unmarshal([]byte(readConfig(t, dir)), &onDisk); err != nil {
		t.Fatal(err)
	}

	reg := onDisk["goPluginInstallRegistry"].(map[string]any)
	if reg["futureNested"].(map[string]any)["mirror"] != "fast" {
		t.Fatalf("lost goPluginInstallRegistry.futureNested: %v", reg)
	}

	bg := onDisk["goWallpaperBackground"].(map[string]any)
	if bg["futureBgTopKey"] != "keep-bg" {
		t.Fatalf("lost futureBgTopKey: %v", bg)
	}
	if bg["wallpaperCrop"].(map[string]any)["cropExtra"] != float64(9) {
		t.Fatalf("lost cropExtra: %v", bg["wallpaperCrop"])
	}
	if bg["wallpaperSources"].([]any)[0].(map[string]any)["sourceExtra"] != true {
		t.Fatalf("lost sourceExtra: %v", bg["wallpaperSources"])
	}
	if bg["wallpaperFavorites"].([]any)[0].(map[string]any)["favExtra"] != "keep-fav" {
		t.Fatalf("lost favExtra: %v", bg["wallpaperFavorites"])
	}

	ap := onDisk["goAppearance"].(map[string]any)
	if ap["futureAppearanceKey"] != "keep-app" {
		t.Fatalf("lost futureAppearanceKey: %v", ap)
	}
	if ap["interface"].(map[string]any)["fontHint"] != "subpixel" || ap["chat"].(map[string]any)["lineHeight"] != 1.4 {
		t.Fatalf("lost nested font keys: %v", ap)
	}
	fam := ap["custom"].([]any)[0].(map[string]any)
	if fam["familyExtra"] != "keep-fam" || fam["light"].(map[string]any)["seedExtra"] != "light-keep" || fam["dark"].(map[string]any)["seedExtra"] != "dark-keep" {
		t.Fatalf("lost custom theme nested unknown fields: %v", fam)
	}

	petMap := onDisk["mygoDesktopPet"].(map[string]any)
	if petMap["futurePetFlag"] != "keep-pet" {
		t.Fatalf("lost futurePetFlag: %v", petMap)
	}
	if petMap["behavior"].(map[string]any)["behaviorExtra"] != float64(7) {
		t.Fatalf("lost behaviorExtra: %v", petMap["behavior"])
	}
	if petMap["position"].(map[string]any)["posExtra"] != "drag-v2" {
		t.Fatalf("lost posExtra: %v", petMap["position"])
	}

	// Clearing optional position deletes "position" while keeping other nested unknown keys.
	pet.Position = nil
	if err := s.SaveDesktopPet(pet); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(readConfig(t, dir)), &onDisk); err != nil {
		t.Fatal(err)
	}
	petMap = onDisk["mygoDesktopPet"].(map[string]any)
	if _, hasPos := petMap["position"]; hasPos {
		t.Fatalf("cleared Position should be omitted: %v", petMap)
	}
	if petMap["futurePetFlag"] != "keep-pet" || petMap["behavior"].(map[string]any)["behaviorExtra"] != float64(7) {
		t.Fatalf("clearing Position dropped other unknown keys: %v", petMap)
	}
}

func TestSaveRemoteGatewayStaleAbortsWithoutDiskTouch(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"port":4000,"remoteMode":"relay","remotePort":6767,"remoteBindAddress":"127.0.0.1"}`)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	beforeBytes := readConfig(t, dir)
	staleExpected := RemoteGatewayPreferences{Mode: "lan", Port: "9999", BindAddress: "0.0.0.0"}
	next := RemoteGatewayPreferences{Mode: "lan", Port: "7788", BindAddress: "127.0.0.1"}
	err = s.SaveRemoteGateway(staleExpected, next)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict-wrapped stale error, got %v", err)
	}
	if got := readConfig(t, dir); got != beforeBytes {
		t.Fatalf("stale SaveRemoteGateway touched disk: %s", got)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".config.lock")); !os.IsNotExist(statErr) {
		t.Fatalf("stale SaveRemoteGateway acquired commit lock before aborting: %v", statErr)
	}
}

func TestAllSavePathsUseGuardedRename(t *testing.T) {
	cases := map[string]func(*Store) error{
		"Update": func(s *Store) error { return s.Update(func(c *Config) { c.Port = 5100 }) },
		"SaveWallpaperBackground": func(s *Store) error {
			return s.SaveWallpaperBackground(wallpapers.Defaults())
		},
		"SaveAppearance": func(s *Store) error {
			return s.SaveAppearance(theme.Defaults())
		},
		"SaveDesktopPet": func(s *Store) error {
			return s.SaveDesktopPet(DesktopPet{Enabled: true, Behavior: DefaultPetBehavior()})
		},
		"SavePageShortcuts": func(s *Store) error {
			return s.SavePageShortcuts(map[string]string{"open": "Ctrl+O"})
		},
		"SaveShortcutDocument": func(s *Store) error {
			return s.SaveShortcutDocument([]byte(`{"version":1}`))
		},
		"SaveWorkspaceNavigationPreferences": func(s *Store) error {
			return s.SaveWorkspaceNavigationPreferences("host", WorkspaceNavigationPreferences{GroupBy: "flat"})
		},
		"UpdatePluginRecovery": func(s *Store) error {
			return s.UpdatePluginRecovery(func(m map[string]any) { m["skipUserPlugins"] = true })
		},
		"PatPet": func(s *Store) error {
			return s.PatPet(time.UnixMilli(1_790_000_000_000))
		},
	}
	for name, save := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, `{"port":4000,"live2dPet":{"enabled":true}}`)
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			// Simulate a transient rename error accompanied by an uncooperative external write:
			// guardedRenameFor must catch the changed baseline before retry attempt 2 and return ErrConflict.
			externalPayload := []byte(`{"port":9999,"external":true}`)
			renames := 0
			s.rename = func(oldpath, newpath string) error {
				renames++
				if err := os.WriteFile(s.path, externalPayload, 0o600); err != nil {
					return err
				}
				return errors.New("fixture sharing violation")
			}
			err = save(s)
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("%s did not detect mid-rename retry conflict: %v", name, err)
			}
			if renames != 1 {
				t.Fatalf("%s retried %d times despite conflict in guardedRename", name, renames)
			}
			if got, _ := os.ReadFile(s.path); !bytes.Equal(got, externalPayload) {
				t.Fatalf("%s overwrote external write inside rename hook: %s", name, got)
			}
		})
	}
}

func TestCommitLockRetriesBriefContentionAndTimesOutCleanly(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"port":4000}`)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	release, err := acquireCommitLock(s.path)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(45 * time.Millisecond)
		release()
	}()
	if err := s.Update(func(c *Config) { c.Port = 4321 }); err != nil {
		t.Fatalf("expected Update to succeed once brief commit lock released: %v", err)
	}
	if s.Get().Port != 4321 {
		t.Fatalf("port = %d, want 4321", s.Get().Port)
	}
}
