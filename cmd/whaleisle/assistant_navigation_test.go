package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/frame"
)

// Compare the navigation gate with real desktop overlay preparation, including
// recovery and stale disabled entries for the immutable Whale built-in. No Host
// or model is started: every config, manifest, overlay and link is owned by t.
func TestAssistantNavigationMatchesDesktopWhaleOverlay(t *testing.T) {
	vendor := t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im", "dsh-whale"} {
		dir := filepath.Join(vendor, name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, enabled := range []bool{false, true} {
		for _, skip := range []bool{false, true} {
			for _, disabled := range []bool{false, true} {
				t.Run(fmt.Sprintf("whale_%t/recovery_%t/list_%t", enabled, skip, disabled), func(t *testing.T) {
					settings := t.TempDir()
					var disabledPlugins []string
					if disabled {
						disabledPlugins = []string{"dsh-whale", "dshwhale", "fixture-user"}
					}
					raw, err := json.Marshal(map[string]any{
						"whaleAssistantEnabled": enabled,
						"pluginRecovery":        map[string]any{"skipUserPlugins": skip},
						"disabledPlugins":       disabledPlugins,
					})
					if err != nil {
						t.Fatal(err)
					}
					if err = os.WriteFile(filepath.Join(settings, config.FileName), raw, 0600); err != nil {
						t.Fatal(err)
					}
					store, err := config.Open(settings)
					if err != nil {
						t.Fatal(err)
					}
					cfg := store.Get()
					if cfg.WhaleAssistantEnabled != enabled || cfg.SkipUserPlugins != skip || !slices.Equal(cfg.DisabledPlugins, disabledPlugins) {
						t.Fatalf("fixture did not load requested accepted config: %+v", cfg)
					}
					profile := t.TempDir()
					if err = os.WriteFile(filepath.Join(profile, "package.json"), []byte(`{"dsh":{"profile":{"bundles":["dsh-whale","dshwhale","fixture-user"]}}}`), 0600); err != nil {
						t.Fatal(err)
					}
					patches, err := harness.Prepare(harness.PrepareOptions{
						VendorDir: vendor, ProfileDir: profile, DshHome: t.TempDir(),
						Whale: cfg.WhaleAssistantEnabled, SkipUserPlugins: cfg.SkipUserPlugins,
						DisabledPlugins: cfg.DisabledPlugins, RemoteSet: true, Remote: false,
					})
					if err != nil {
						t.Fatal(err)
					}
					hasPatch := func(name string) bool {
						for _, patch := range patches {
							if filepath.Base(patch) == name {
								return true
							}
						}
						return false
					}
					mounted := hasPatch("desktop-dsh-whale.patch.yml")
					if mounted != enabled || hasPatch("desktop-session-search.patch.yml") == skip {
						t.Fatalf("Whale/recovery overlay policy drift: %v", patches)
					}
					var manifest struct {
						DSH struct {
							Profile struct {
								Bundles []string `json:"bundles"`
							} `json:"profile"`
						} `json:"dsh"`
					}
					data, err := os.ReadFile(filepath.Join(profile, "package.json"))
					if err != nil {
						t.Fatal(err)
					}
					if err = json.Unmarshal(data, &manifest); err != nil {
						t.Fatal(err)
					}
					bundles := manifest.DSH.Profile.Bundles
					if !slices.Contains(bundles, "dsh-whale") || !slices.Contains(bundles, "dshwhale") || slices.Contains(bundles, "fixture-user") == disabled {
						t.Fatalf("immutable built-in or disabled user bundle behavior drift: %v", bundles)
					}
					s := &mainState{cfgStore: store}
					for _, phase := range []harness.State{harness.Stopped, harness.Starting, harness.Ready, harness.Failed} {
						s.bootStatus.State = phase
						calls := 0
						m := frame.Model{OnOpenAssistant: func() { calls++ }, OnOpenSettings: func() {}}
						s.gateAssistantNavigation(&m)
						if (m.OnOpenAssistant != nil) != mounted || m.OnOpenSettings == nil {
							t.Fatalf("phase=%v navigation and overlay disagree; mounted=%v", phase, mounted)
						}
						if m.OnOpenAssistant != nil {
							m.OnOpenAssistant()
							if calls != 1 {
								t.Fatal("gate replaced the existing assistant route")
							}
						}
					}
				})
			}
		}
	}
}

func TestAssistantNavigationUsesCommittedConfigAndLeavesSettingsRoute(t *testing.T) {
	s := &mainState{}
	m := frame.Model{OnOpenAssistant: func() {}, OnOpenSettings: func() {}}
	s.gateAssistantNavigation(&m)
	if m.OnOpenAssistant != nil || m.OnOpenSettings == nil {
		t.Fatal("missing config must hide navigation while retaining the settings route")
	}
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.cfgStore = store
	for _, enabled := range []bool{true, false, true} {
		if err = store.Update(func(c *config.Config) { c.WhaleAssistantEnabled = enabled }); err != nil {
			t.Fatal(err)
		}
		m.OnOpenAssistant = func() {}
		s.gateAssistantNavigation(&m)
		if (m.OnOpenAssistant != nil) != enabled || m.OnOpenSettings == nil {
			t.Fatal("gate did not read the current accepted preference")
		}
		m.OnOpenAssistant = nil
		s.gateAssistantNavigation(&m)
		if m.OnOpenAssistant != nil {
			t.Fatal("gate invented a route for a caller without one")
		}
	}
	cancelled := errors.New("cancelled preference change")
	if err = store.UpdateChecked(func(c *config.Config) error {
		c.WhaleAssistantEnabled = false
		return cancelled
	}); !errors.Is(err, cancelled) {
		t.Fatalf("fixture cancellation: %v", err)
	}
	m.OnOpenAssistant = func() {}
	s.gateAssistantNavigation(&m)
	if m.OnOpenAssistant == nil {
		t.Fatal("uncommitted preference candidate hid the accepted enabled route")
	}
}
