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
	if len(patches) != 9 { // 4 builtin + remote + plugin views + native contribution UI + proactive + market
		t.Fatalf("got %d patches, want 9: %v", len(patches), patches)
	}
	joined := strings.Join(patches, "\n")
	if !strings.Contains(joined, "desktop-dsh-market.patch.yml") {
		t.Fatalf("missing market overlay: %s", joined)
	}
	for _, p := range patches {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
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
	if len(patches) != 10 { // 4 builtin + dshbot + whale + plugin views + native UI + proactive + market, no remote
		t.Fatalf("got %d patches, want 10", len(patches))
	}
	joined := strings.Join(patches, "\n")
	if !strings.Contains(joined, "desktop-dshbot.patch.yml") || !strings.Contains(joined, "desktop-dsh-whale.patch.yml") {
		t.Fatalf("missing optional overlays: %s", joined)
	}
	if strings.Contains(joined, "desktop-dsh-remote.patch.yml") {
		t.Fatal("remote should be off")
	}
}

func TestOverlayYAMLQuotesScopedName(t *testing.T) {
	got := overlayYAML("xmanrui-dsh-im", "@xmanrui/dsh-im")
	if !strings.Contains(got, `name: "@xmanrui/dsh-im"`) {
		t.Fatalf("got %q", got)
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
