package startuprec

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestOfflinePluginFixture uses only a temp profile and injected uninstall.
func TestOfflinePluginFixture(t *testing.T) {
	profile := t.TempDir()
	file := filepath.Join(profile, "package.json")
	original := `{"dependencies":{"demo":"1.0.0","@deepseek-ai/dsh-base":"1"},"dsh":{"profile":{"bundles":["@deepseek-ai/dsh-base","demo"],"keep":true}},"keep":"yes"}`
	if err := os.WriteFile(file, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	disabled := []string{}
	p := OfflinePlugins{ProfileDir: profile, LoadDisabled: func() []string { return slices.Clone(disabled) }, SaveDisabled: func(v []string) error { disabled = slices.Clone(v); return nil }}
	rows, err := p.Inventory()
	if err != nil || len(rows) != 2 {
		t.Fatalf("inventory %v %v", rows, err)
	}
	if err = p.SetEnabled("demo", false); err != nil {
		t.Fatal(err)
	}
	rows, _ = p.Inventory()
	if rows[1].Enabled || !slices.Contains(disabled, "demo") {
		t.Fatalf("disable not reflected: %v %v", rows, disabled)
	}
	d := Diagnose([]string{"Cannot find package 'demo'"}, "", []Plugin{{Name: "demo", Enabled: true, Mutable: true}}, true)
	if !d.CanRepair {
		t.Fatal("offline inventory must permit suspects")
	}
	if err = p.SetEnabled("demo", true); err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 0 {
		t.Fatal(disabled)
	}
	if err = p.SetEnabled("@deepseek-ai/dsh-base", false); err == nil {
		t.Fatal("template disabled")
	}
	if err = p.SetEnabled("demo", false); err != nil {
		t.Fatal(err)
	}
	p.Uninstall = func(string) error { return errors.New("CLI failed") }
	if err = p.Remove("demo"); err == nil || !slices.Contains(disabled, "demo") {
		t.Fatal("failed uninstall cleared disabled entry")
	}
	p.Uninstall = func(name string) error {
		if name != "demo" {
			t.Fatal(name)
		}
		return nil
	}
	if err = p.Remove("demo"); err != nil || len(disabled) != 0 {
		t.Fatalf("remove %v %v", err, disabled)
	}
	data, _ := os.ReadFile(file)
	var saved map[string]any
	if err = json.Unmarshal(data, &saved); err != nil || saved["keep"] != "yes" {
		t.Fatalf("unrelated fields lost: %s", data)
	}
}

func TestOfflinePluginsSetEnabledSaveDisabledFailureLeavesNoSplitState(t *testing.T) {
	for _, tc := range []struct {
		name           string
		initialBundles string
		initialDis     []string
		enable         bool
		wantEnabled    bool
	}{
		{
			name:           "disable fails",
			initialBundles: `["@deepseek-ai/dsh-base","demo"]`,
			initialDis:     nil,
			enable:         false,
			wantEnabled:    true,
		},
		{
			name:           "enable fails",
			initialBundles: `["@deepseek-ai/dsh-base"]`,
			initialDis:     []string{"demo"},
			enable:         true,
			wantEnabled:    false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := t.TempDir()
			file := filepath.Join(profile, "package.json")
			original := `{"dependencies":{"demo":"1.0.0","@deepseek-ai/dsh-base":"1"},"dsh":{"profile":{"bundles":` + tc.initialBundles + `}}}`
			if err := os.WriteFile(file, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			disabled := slices.Clone(tc.initialDis)
			saveErr := errors.New("synthetic config write failure")
			p := OfflinePlugins{
				ProfileDir:   profile,
				LoadDisabled: func() []string { return slices.Clone(disabled) },
				SaveDisabled: func([]string) error { return saveErr },
			}
			if err := p.SetEnabled("demo", tc.enable); !errors.Is(err, saveErr) {
				t.Fatalf("expected save failure, got %v", err)
			}
			gotBytes, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotBytes) != original {
				t.Fatalf("package.json mutated despite SaveDisabled failure:\ngot:  %s\nwant: %s", gotBytes, original)
			}
			rows, err := p.Inventory()
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range rows {
				if r.Name == "demo" && r.Enabled != tc.wantEnabled {
					t.Fatalf("split state in Inventory: got enabled=%v, want %v", r.Enabled, tc.wantEnabled)
				}
			}
		})
	}
}
