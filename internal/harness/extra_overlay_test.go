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

func TestRouteAOverlayDisablesClientHMR(t *testing.T) {
	vendor := t.TempDir()
	profile := t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im"} {
		dir := filepath.Join(vendor, name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644)
	}
	patches, err := Prepare(PrepareOptions{VendorDir: vendor, ProfileDir: profile, RemoteSet: true, Remote: false})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(patches, "\n")
	if !strings.Contains(joined, routeAOverlayFile) {
		t.Fatalf("missing route-a overlay in %s", joined)
	}
	data, err := os.ReadFile(filepath.Join(profile, "desktop-plugins", "route-a", routeAOverlayFile))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "- id: client-hmr") || !strings.Contains(text, "disabled: true") {
		t.Fatalf("unexpected route-a overlay body:\n%s", text)
	}
}

func TestRouteAOverlayNeverDisablesShippedCapabilities(t *testing.T) {
	profile := t.TempDir()
	harnessRoot := t.TempDir()

	// A harness with neither plugin installed still must not turn either plugin
	// off: the released package asserts the pairs at packaging time, and a
	// silent disable here is what let a paid-for capability ship dead.
	for _, stage := range []string{"absent", "loader-only", "complete"} {
		switch stage {
		case "loader-only":
			librePkg := filepath.Join(harnessRoot, "node_modules", "@deepseek-ai", "libreoffice-kit")
			if err := os.MkdirAll(librePkg, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(librePkg, "package.json"), []byte(`{"name":"@deepseek-ai/libreoffice-kit"}`), 0o644); err != nil {
				t.Fatal(err)
			}
		case "complete":
			sherpaPkg := filepath.Join(harnessRoot, "packages", "experimental", "speech-to-text-sensevoice", "node_modules", "sherpa-onnx-node")
			if err := os.MkdirAll(sherpaPkg, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(sherpaPkg, "package.json"), []byte(`{"name":"sherpa-onnx-node"}`), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		patch, err := writeRouteAHostOverlay(PrepareOptions{ProfileDir: profile, HarnessRoot: harnessRoot})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(patch)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, "- id: client-hmr") {
			t.Fatalf("%s: client-hmr must always be disabled:\n%s", stage, text)
		}
		if strings.Contains(text, "- id: office-to-pdf") || strings.Contains(text, "- id: speech-to-text-sensevoice") {
			t.Fatalf("%s: a shipped capability must never be disabled by the overlay:\n%s", stage, text)
		}
	}
}

func TestHostSafetyOverlayMountsObservationPolicy(t *testing.T) {
	vendor := t.TempDir()
	profile := t.TempDir()
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im"} {
		dir := filepath.Join(vendor, name)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644)
	}
	patches, err := Prepare(PrepareOptions{VendorDir: vendor, ProfileDir: profile, RemoteSet: true, Remote: false})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(patches, "\n"), hostSafetyOverlayFile) {
		t.Fatalf("missing host-safety overlay: %v", patches)
	}
	data, err := os.ReadFile(filepath.Join(profile, "desktop-plugins", "host-safety", hostSafetyOverlayFile))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "- id: fs-observation-policy") || !strings.Contains(body, "disabled: false") {
		t.Fatalf("overlay body: %s", body)
	}
	recovery, err := Prepare(PrepareOptions{VendorDir: vendor, ProfileDir: t.TempDir(), SkipUserPlugins: true, RemoteSet: true, Remote: false})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(recovery, "\n"), hostSafetyOverlayFile) {
		t.Fatalf("recovery start must still mount the observation policy: %v", recovery)
	}
}
