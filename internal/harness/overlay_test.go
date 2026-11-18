package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareWritesOverlaysAndJunctions(t *testing.T) {
	vendor := t.TempDir()
	profile := t.TempDir()
	for _, name := range []string{
		"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im", "dsh-remote",
	} {
		dir := filepath.Join(vendor, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		pkg := name
		if name == "dsh-im" {
			pkg = "@xmanrui/dsh-im"
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+pkg+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	patches, err := Prepare(PrepareOptions{VendorDir: vendor, ProfileDir: profile, RemoteSet: true, Remote: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 9 { // 4 builtin + remote + plugin views + route-a + proactive + host-safety
		t.Fatalf("got %d patches, want 9: %v", len(patches), patches)
	}
	joined := strings.Join(patches, "\n")
	if strings.Contains(joined, "desktop-dsh-market.patch.yml") || strings.Contains(joined, "pluginui") {
		t.Fatalf("legacy pluginui/market overlay should be removed: %s", joined)
	}
	if !strings.Contains(joined, routeAOverlayFile) {
		t.Fatalf("missing route-a overlay: %s", joined)
	}
	for _, p := range patches {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(p, routeAOverlayFile) {
			if !strings.Contains(string(data), "- id: client-hmr") || !strings.Contains(string(data), "disabled: true") {
				t.Fatalf("route-a overlay unexpected body: %s", data)
			}
			continue
		}
		if strings.HasSuffix(p, hostSafetyOverlayFile) {
			if !strings.Contains(string(data), "- id: fs-observation-policy") || !strings.Contains(string(data), "disabled: false") {
				t.Fatalf("host-safety overlay unexpected body: %s", data)
			}
			continue
		}
		if !strings.Contains(string(data), "- insert:") {
			t.Fatalf("overlay missing insert: %s", p)
		}
	}
	link := filepath.Join(profile, "node_modules", "dsh-task-control")
	if _, err := os.Stat(filepath.Join(link, "package.json")); err != nil {
		t.Fatalf("junction missing: %v", err)
	}
	scoped := filepath.Join(profile, "node_modules", "@xmanrui", "dsh-im", "package.json")
	if _, err := os.Stat(scoped); err != nil {
		t.Fatalf("scoped junction missing: %v", err)
	}
}

func TestPrepareOptionalPlugins(t *testing.T) {
	vendor := t.TempDir()
	profile := t.TempDir()
	for _, name := range []string{
		"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im", "dshbot", "dsh-whale",
	} {
		dir := filepath.Join(vendor, name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644)
	}
	patches, err := Prepare(PrepareOptions{
		VendorDir: vendor, ProfileDir: profile,
		Dshbot: true, Whale: true, RemoteSet: true, Remote: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 10 { // 4 builtin + dshbot + whale + plugin views + route-a + proactive + host-safety, no remote
		t.Fatalf("got %d patches, want 10", len(patches))
	}
	joined := strings.Join(patches, "\n")
	if !strings.Contains(joined, "desktop-dshbot.patch.yml") || !strings.Contains(joined, "desktop-dsh-whale.patch.yml") {
		t.Fatalf("missing optional overlays: %s", joined)
	}
	if !strings.Contains(joined, hostSafetyOverlayFile) {
		t.Fatalf("missing host-safety overlay: %s", joined)
	}
	if strings.Contains(joined, "desktop-dsh-remote.patch.yml") {
		t.Fatal("remote should be off")
	}
}

func TestOverlayYAMLQuotesScopedName(t *testing.T) {
	got := overlayYAML(DesktopPlugin{InsertID: "xmanrui-dsh-im", Package: "@xmanrui/dsh-im"})
	if !strings.Contains(got, `name: "@xmanrui/dsh-im"`) {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "disabled:") {
		t.Fatalf("an ungated plugin must not carry a gate: %q", got)
	}
	gated := overlayYAML(DesktopPlugin{InsertID: "dsh-deepseek-tools", Package: "dsh-deepseek-tools", GateEnv: "DSHD_DEEPSEEK_TOOLS_ENABLED"})
	if !strings.Contains(gated, `disabled: !!js "process.env.DSHD_DEEPSEEK_TOOLS_ENABLED !== '1'"`) {
		t.Fatalf("gate row = %q", gated)
	}
}

func TestPrepareACPRequiresCompleteT3Package(t *testing.T) {
	vendor, profile := t.TempDir(), t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im", "dsh-acp"} {
		dir := filepath.Join(vendor, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	opts := PrepareOptions{VendorDir: vendor, ProfileDir: profile, RemoteSet: true}
	if _, err := Prepare(opts); err == nil {
		t.Fatal("accepted ACP without its T3 bundle")
	}
	for _, name := range []string{"lib/index.js", "lib/presets.js", "lib/t3-runtime.js", "lib/t3-session.js", "lib/t3-auth.js", "lib/mcp-server.js", "lib/vendor/t3.cjs", "lib/vendor/t3-mcp-bridge.cjs", "client/client.js"} {
		file := filepath.Join(vendor, "dsh-acp", name)
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("// owned fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	patches, err := Prepare(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(patches, "\n"), "desktop-acp.patch.yml") {
		t.Fatal("ACP not mounted")
	}
	if _, err := os.Stat(filepath.Join(profile, "node_modules", "dsh-acp-agents", "lib", "vendor", "t3.cjs")); err != nil {
		t.Fatal(err)
	}
}

// The DeepSeek application API row arrives through --patch, gated on the flag
// launch.go sets, and the profile junction resolves the package it names.
func TestPrepareMountsDeepSeekToolsBehindItsFlag(t *testing.T) {
	vendor, profile := t.TempDir(), t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im", "dsh-deepseek-tools"} {
		dir := filepath.Join(vendor, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		pkg := name
		if name == "dsh-im" {
			pkg = "@xmanrui/dsh-im"
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+pkg+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	patches, err := Prepare(PrepareOptions{VendorDir: vendor, ProfileDir: profile, RemoteSet: true, Remote: false})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(profile, "node_modules", "dsh-deepseek-tools", "package.json")); err != nil {
		t.Fatalf("profile junction missing: %v", err)
	}
	row := filepath.Join(profile, "desktop-plugins", "dsh-deepseek-tools", "desktop-deepseek-tools.patch.yml")
	body, err := os.ReadFile(row)
	if err != nil {
		t.Fatalf("overlay row missing: %v", err)
	}
	if !strings.Contains(string(body), `name: "dsh-deepseek-tools"`) {
		t.Fatalf("row does not name the package it mounts: %s", body)
	}
	if !strings.Contains(string(body), `process.env.DSHD_DEEPSEEK_TOOLS_ENABLED !== '1'`) {
		t.Fatalf("row is not gated: %s", body)
	}
	found := false
	for _, patch := range patches {
		if patch == row {
			found = true
		}
	}
	if !found {
		t.Fatalf("gated row not among the launch patches: %v", patches)
	}
}
