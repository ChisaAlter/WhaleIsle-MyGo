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

func TestRouteAOverlaySlimmedRuntimeDisablesMissingPlugins(t *testing.T) {
	profile := t.TempDir()
	harnessRoot := t.TempDir()

	// 1. Missing both: should disable client-hmr, office-to-pdf, speech-to-text-sensevoice
	patch, err := writeRouteAHostOverlay(PrepareOptions{ProfileDir: profile, HarnessRoot: harnessRoot})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(patch)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "- id: client-hmr") ||
		!strings.Contains(text, "- id: office-to-pdf") ||
		!strings.Contains(text, "- id: speech-to-text-sensevoice") {
		t.Fatalf("expected all 3 disabled when packages absent, got:\n%s", text)
	}

	// 2. Present libreoffice-kit: only client-hmr and speech-to-text-sensevoice disabled
	librePkg := filepath.Join(harnessRoot, "node_modules", "@deepseek-ai", "libreoffice-kit")
	if err := os.MkdirAll(librePkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(librePkg, "package.json"), []byte(`{"name":"@deepseek-ai/libreoffice-kit"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	patch, err = writeRouteAHostOverlay(PrepareOptions{ProfileDir: profile, HarnessRoot: harnessRoot})
	if err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(patch)
	if err != nil {
		t.Fatal(err)
	}
	text = string(data)
	if strings.Contains(text, "- id: office-to-pdf") {
		t.Fatalf("office-to-pdf should not be disabled when libreoffice-kit is present:\n%s", text)
	}
	if !strings.Contains(text, "- id: speech-to-text-sensevoice") {
		t.Fatalf("speech-to-text-sensevoice should be disabled when sherpa is absent:\n%s", text)
	}

	// 3. Present sherpa-onnx-node in packages/experimental/speech-to-text-sensevoice/node_modules: neither disabled except client-hmr
	sherpaPkg := filepath.Join(harnessRoot, "packages", "experimental", "speech-to-text-sensevoice", "node_modules", "sherpa-onnx-node")
	if err := os.MkdirAll(sherpaPkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sherpaPkg, "package.json"), []byte(`{"name":"sherpa-onnx-node"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	patch, err = writeRouteAHostOverlay(PrepareOptions{ProfileDir: profile, HarnessRoot: harnessRoot})
	if err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(patch)
	if err != nil {
		t.Fatal(err)
	}
	text = string(data)
	if strings.Contains(text, "- id: office-to-pdf") || strings.Contains(text, "- id: speech-to-text-sensevoice") {
		t.Fatalf("neither plugin should be disabled when both packages are present:\n%s", text)
	}
	if !strings.Contains(text, "- id: client-hmr") {
		t.Fatalf("client-hmr must always be disabled:\n%s", text)
	}
}
