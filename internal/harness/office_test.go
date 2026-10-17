package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureOfficeOverlaySkipsWithoutPayload(t *testing.T) {
	t.Setenv("DSH_PRIMARY_RUNTIME", "")
	profile := t.TempDir()
	p, err := EnsureOfficeOverlay(PrepareOptions{ProfileDir: profile, DshHome: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if p != "" {
		t.Fatalf("expected skip, got %s", p)
	}
}

func TestEnsureOfficeOverlayWritesWhenPresent(t *testing.T) {
	payload, harnessRoot := syntheticOfficePayload(t)
	t.Setenv("DSH_PRIMARY_RUNTIME", payload)
	profile := t.TempDir()
	home := t.TempDir()
	p, err := EnsureOfficeOverlay(PrepareOptions{
		ProfileDir:  profile,
		DshHome:     home,
		HarnessRoot: harnessRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p == "" {
		t.Fatal("expected overlay path")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{"workspace-dependencies", "skill-office", "dsh-primary-runtime"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

// TestPrepareIncludesOfficeOverlayWhenPayloadPresent covers the Prepare →
// extraOverlays → EnsureOfficeOverlay path used at harness start (not just
// EnsureOfficeOverlay in isolation).
func TestPrepareIncludesOfficeOverlayWhenPayloadPresent(t *testing.T) {
	payload, harnessRoot := syntheticOfficePayload(t)
	t.Setenv("DSH_PRIMARY_RUNTIME", payload)

	vendor := t.TempDir()
	profile := t.TempDir()
	home := t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im"} {
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

	patches, err := Prepare(PrepareOptions{
		VendorDir:   vendor,
		ProfileDir:  profile,
		DshHome:     home,
		HarnessRoot: harnessRoot,
		RemoteSet:   true,
		Remote:      false,
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(patches, "\n")
	if !strings.Contains(joined, officeOverlayFile) {
		t.Fatalf("Prepare patches missing office overlay %q: %s", officeOverlayFile, joined)
	}
	officePath := filepath.Join(profile, "desktop-plugins", officeOverlayDir, officeOverlayFile)
	data, err := os.ReadFile(officePath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{"workspace-dependencies", "skill-office", "dsh-primary-runtime", filepath.ToSlash(payload)} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in office overlay:\n%s", want, body)
		}
	}
}

func syntheticOfficePayload(t *testing.T) (payload, harnessRoot string) {
	t.Helper()
	payload = t.TempDir()
	if err := os.WriteFile(filepath.Join(payload, "runtime.json"), []byte(`{"platform":"x","arch":"y","payloadDigest":"d"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	nodeDir := filepath.Join(payload, "dependencies", "node", "bin")
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(nodeDir, "node"), []byte(""), 0o755)
	_ = os.WriteFile(filepath.Join(nodeDir, "node.exe"), []byte(""), 0o755)
	harnessRoot = t.TempDir()
	cli := filepath.Join(harnessRoot, "node_modules", "@deepseek-ai", "libreoffice-kit", "lib", "cli.js")
	if err := os.MkdirAll(filepath.Dir(cli), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	return payload, harnessRoot
}
