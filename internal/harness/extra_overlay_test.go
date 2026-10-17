package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionSearchOverlayOnFullStart(t *testing.T) {
	vendor := t.TempDir()
	profile := t.TempDir()
	home := t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im"} {
		dir := filepath.Join(vendor, name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644)
	}
	patches, err := Prepare(PrepareOptions{
		VendorDir: vendor, ProfileDir: profile, DshHome: home,
		RemoteSet: true, Remote: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(patches, "\n")
	if !strings.Contains(joined, sessionSearchOverlayFile) {
		t.Fatalf("missing session-search overlay: %s", joined)
	}
	data, err := os.ReadFile(filepath.Join(profile, "desktop-plugins", "session-search", sessionSearchOverlayFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "session-query.sqlite") || !strings.Contains(string(data), "openAt: first-search") {
		t.Fatalf("overlay body: %s", data)
	}
}

func TestSessionSearchOmittedWhenSkipping(t *testing.T) {
	vendor := t.TempDir()
	profile := t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im"} {
		dir := filepath.Join(vendor, name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644)
	}
	patches, err := Prepare(PrepareOptions{
		VendorDir: vendor, ProfileDir: profile, DshHome: t.TempDir(),
		SkipUserPlugins: true, RemoteSet: true, Remote: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(patches, "\n"), sessionSearchOverlayFile) {
		t.Fatal("session-search overlay should be omitted on skip")
	}
}
