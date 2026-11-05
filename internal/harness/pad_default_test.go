package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/desktop/openui"
)

func padFixture(t *testing.T) PrepareOptions {
	t.Helper()
	o := PrepareOptions{VendorDir: t.TempDir(), ProfileDir: t.TempDir()}
	root := filepath.Join(o.VendorDir, "dsh-pad")
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"package.json": `{"name":"dsh-pad","version":"0.3.1"}`, "lib/index.js": "export function apply(){}", "lib/client.js": "window.__ModuleLoader__.load({})", "cordis.patch.yml": "- insert: []\n"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return o
}
func TestPadDefaultHonorsUserSelectionAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name, manifest       string
		skip, disabled, want bool
	}{{"fresh", `{}`, false, false, true}, {"user-bundle", `{"dependencies":{"dsh-pad":"0.3.1"},"dsh":{"profile":{"bundles":["dsh-pad"]}}}`, false, false, false}, {"user-disabled", `{"dependencies":{"dsh-pad":"0.3.1"},"dsh":{"profile":{"bundles":[]}}}`, false, false, false}, {"recovery-shipped", `{"dependencies":{"dsh-pad":"0.3.1"},"dsh":{"profile":{"bundles":["dsh-pad"]}}}`, true, false, true}, {"explicit-disabled", `{}`, true, true, false}} {
		t.Run(tc.name, func(t *testing.T) {
			o := padFixture(t)
			o.SkipUserPlugins = tc.skip
			if tc.disabled {
				o.DisabledPlugins = []string{"dsh-pad"}
			}
			manifest := filepath.Join(o.ProfileDir, "package.json")
			if err := os.WriteFile(manifest, []byte(tc.manifest), 0600); err != nil {
				t.Fatal(err)
			}
			patch, err := PreparePadDefault(o)
			if err != nil {
				t.Fatal(err)
			}
			if (patch != "") != tc.want {
				t.Fatalf("patch %q want %v", patch, tc.want)
			}
			after, _ := os.ReadFile(manifest)
			if string(after) != tc.manifest {
				t.Fatal("Pad rewrote user manifest")
			}
			if tc.want {
				raw, _ := os.ReadFile(patch)
				if !strings.Contains(string(raw), "file:///") || !strings.Contains(string(raw), "updateCheck: false") {
					t.Fatal(string(raw))
				}
				if _, err := os.Stat(filepath.Join(o.ProfileDir, "node_modules", "whaleisle-desktop-pad", "lib", "index.js")); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestPadDefaultMissingCustomCompositionRemainsUsable(t *testing.T) {
	patch, err := PreparePadDefault(PrepareOptions{VendorDir: t.TempDir(), ProfileDir: t.TempDir()})
	if err != nil || patch != "" {
		t.Fatal(patch, err)
	}
}
func TestManagedWhaleUIValidatorAndDefaultLayerEnvironment(t *testing.T) {
	home := t.TempDir()
	path, err := prepareWhaleUIHost(home)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != string(openui.WhaleHostValidator()) {
		t.Fatal("validator differs", err)
	}
	t.Setenv("DSHD_PROFILE_DEFAULT_PATCH", "foreign")
	t.Setenv("DSHD_WHALE_UI_HOST", "foreign-validator")
	managed := strings.Join(childEnv(LaunchOptions{DesktopDefaults: true, ProfileDefaultPatch: "owned-default", WhaleUIHost: path}), "\n")
	for _, required := range []string{"DSHD_PROFILE_DEFAULT_PATCH=owned-default", "DSHD_WHALE_UI_ENABLED=1", "DSHD_WHALE_UI_HOST=" + path} {
		if !strings.Contains(managed, required) {
			t.Fatal(required)
		}
	}
	if strings.Contains(managed, "=foreign") {
		t.Fatal("inherited desktop env overrode owned composition")
	}
	external := strings.Join(childEnv(LaunchOptions{}), "\n")
	if !strings.Contains(external, "DSHD_PROFILE_DEFAULT_PATCH=foreign") || !strings.Contains(external, "DSHD_WHALE_UI_HOST=foreign-validator") {
		t.Fatal("changed configured launcher contract")
	}
}
