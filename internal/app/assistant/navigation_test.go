package assistantctl

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
					phase := harness.Stopped
					svc := NewService(HostDeps{
						Config:       func() *config.Store { return store },
						BootEndpoint: func() (harness.Endpoint, harness.State, int) { return harness.Endpoint{}, phase, 1 },
					})
					for _, p := range []harness.State{harness.Stopped, harness.Starting, harness.Ready, harness.Failed} {
						phase = p
						allowed := svc.GateNavigation(true)
						if allowed != mounted {
							t.Fatalf("phase=%v navigation and overlay disagree; mounted=%v allowed=%v", phase, mounted, allowed)
						}
						if svc.GateNavigation(false) {
							t.Fatal("gate invented a route when caller had none")
						}
					}
				})
			}
		}
	}
}

func TestAssistantNavigationUsesCommittedConfigAndLeavesSettingsRoute(t *testing.T) {
	var cfgStore *config.Store
	svc := NewService(HostDeps{
		Config: func() *config.Store { return cfgStore },
	})
	if svc.GateNavigation(true) {
		t.Fatal("missing config must hide navigation")
	}
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfgStore = store
	for _, enabled := range []bool{true, false, true} {
		if err = store.Update(func(c *config.Config) { c.WhaleAssistantEnabled = enabled }); err != nil {
			t.Fatal(err)
		}
		if svc.GateNavigation(true) != enabled {
			t.Fatal("gate did not read the current accepted preference")
		}
		if svc.GateNavigation(false) {
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
	if !svc.GateNavigation(true) {
		t.Fatal("uncommitted preference candidate hid the accepted enabled route")
	}
}
