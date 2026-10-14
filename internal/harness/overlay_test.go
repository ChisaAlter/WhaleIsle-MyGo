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
	if len(patches) != 6 { // 4 builtin + remote + market overlay
		t.Fatalf("got %d patches, want 6: %v", len(patches), patches)
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
	if len(patches) != 7 { // 4 builtin + dshbot + whale + market, no remote
		t.Fatalf("got %d patches, want 7", len(patches))
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
