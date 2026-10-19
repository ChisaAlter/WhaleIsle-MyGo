package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/paths"
)

func writeSessionRuntimeRegistry(t *testing.T, root string, eventTypes []string) {
	t.Helper()
	encoded, err := json.Marshal(eventTypes)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "packages", "core", "session", "lib", "index.js")
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, append(append([]byte("const KNOWN_SESSION_EVENT_TYPES = new Set("), encoded...), []byte(");")...), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSessionRuntimeChecksBuiltVocabularyRatherThanFormatOrSource(t *testing.T) {
	root := t.TempDir()
	writeSessionRuntimeRegistry(t, root, []string{"user/message", "assistant/message"})
	// A new TS snapshot alongside stale built JS must not certify the reader.
	source := filepath.Join(root, "packages", "core", "session", "src", "known-event-types.ts")
	if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte(strings.Join(desktopSessionEventTypes, "\n")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSessionRuntimeForLaunch(root); err == nil || !strings.Contains(err.Error(), "older than") {
		t.Fatalf("stale built reader accepted: %v", err)
	}
	writeSessionRuntimeRegistry(t, root, desktopSessionEventTypes)
	if err := ValidateSessionRuntimeForLaunch(root); err != nil {
		t.Fatal(err)
	}
	writeSessionRuntimeRegistry(t, root, desktopSessionEventTypes[:2])
	if err := ValidateSessionRuntimeForLaunch(root); err == nil {
		t.Fatal("partial event vocabulary accepted")
	}
}

func TestResolveRefusesStaleVendorAndConfiguredSource(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deepseek-harness")
	writeSessionRuntimeRegistry(t, root, []string{"user/message"})
	bin := filepath.Join(root, "apps", "cli", "lib", "bin.js")
	if err := os.MkdirAll(filepath.Dir(bin), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("// inert"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []LaunchOptions{{HarnessRoot: root}, {DshBin: bin}} {
		if _, err := Resolve(opts, 3080); err == nil || !strings.Contains(err.Error(), "older than") {
			t.Fatalf("stale source selection accepted: %v", err)
		}
	}
}

func TestPlanStartRejectsStaleReaderBeforePreparingSharedProfile(t *testing.T) {
	base := t.TempDir()
	userData := filepath.Join(base, "untouched-user-data")
	t.Setenv(paths.EnvUserData, userData)
	root := filepath.Join(base, "vendor", "deepseek-harness")
	writeSessionRuntimeRegistry(t, root, []string{"user/message"})
	profile := filepath.Join(userData, "dsh-home", "profiles", "web")
	_, err := PlanStart(LaunchOptions{HarnessRoot: root}, PrepareOptions{ProfileDir: profile}, 3080)
	if err == nil || !strings.Contains(err.Error(), "older than") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(userData); !os.IsNotExist(err) {
		t.Fatalf("failed preflight touched shared profile: %v", err)
	}
}

func TestSessionRuntimeManifestCannotWaiveReadiness(t *testing.T) {
	base := t.TempDir()
	vendor := filepath.Join(base, "vendor")
	if err := os.MkdirAll(filepath.Join(base, "node"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(vendor, 0700); err != nil {
		t.Fatal(err)
	}
	nodeBytes := []byte("inert Node fixture")
	if err := os.WriteFile(filepath.Join(base, "node", "node.exe"), nodeBytes, 0600); err != nil {
		t.Fatal(err)
	}
	nodeSum := sha256.Sum256(nodeBytes)
	manifest := map[string]any{
		"schemaVersion": 1, "cliVersion": "0.2.1-rc.2",
		"harnessPin":         map[string]string{"repo": "https://github.com/deepseek-ai/deepseek-harness.git", "ref": "dsh-v0.2.1-rc.2", "sha": sessionRuntimeHarnessCommit, "npm": "0.2.1-rc.2"},
		"requiredEventTypes": desktopSessionEventTypes,
		"safetyPatch":        rc2SessionRuntimeSafety,
		"node":               map[string]string{"path": "node/node.exe", "sha256": hex.EncodeToString(nodeSum[:])},
	}
	write := func() {
		t.Helper()
		raw, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(vendor, sessionRuntimeManifestName), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	root := filepath.Join(vendor, "deepseek-harness")
	if err := ValidateCurrentHostForLaunch(root); err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("self-declared manifest accepted without real artifacts: %v", err)
	}
	manifest["safetyPatch"] = sessionRuntimeSafety{ID: rc2SessionRuntimeSafety.ID}
	write()
	if err := ValidateCurrentHostForLaunch(root); err == nil || !strings.Contains(err.Error(), "visual-readiness") {
		t.Fatalf("missing readiness patch accepted: %v", err)
	}
}

func TestSessionRuntimeNodeOverrideMustMatchSelectedAssembly(t *testing.T) {
	vendor := t.TempDir()
	bytes := []byte("inert bundled Node fixture")
	sum := sha256.Sum256(bytes)
	manifest, _ := json.Marshal(map[string]any{"node": map[string]string{"sha256": hex.EncodeToString(sum[:])}})
	if err := os.WriteFile(filepath.Join(vendor, sessionRuntimeManifestName), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	node := filepath.Join(t.TempDir(), "explicit-node.exe")
	if err := os.WriteFile(node, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateSessionRuntimeNode(vendor, node); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(node, []byte("different inert Node"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateSessionRuntimeNode(vendor, node); err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("mismatched Node accepted: %v", err)
	}
}
