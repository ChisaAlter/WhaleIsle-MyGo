package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// installedProofFixture writes a temp vendor tree shaped like the staged rc.2
// runtime: the session manifest with its five hashed artifacts, the readiness
// record, the gated visual-replies bundle and the dsh-project package. The pin
// is overridden so the fixture's inert bytes can stand in for the release
// bundle; the production pins are restored after the test.
func installedProofFixture(t *testing.T) (vendor string, safety sessionRuntimeSafety, manifest map[string]any) {
	t.Helper()
	previous := rc2SessionRuntimeSafety
	t.Cleanup(func() { rc2SessionRuntimeSafety = previous })

	base := t.TempDir()
	vendor = filepath.Join(base, "vendor")
	if err := os.MkdirAll(filepath.Join(base, "node"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(vendor, 0700); err != nil {
		t.Fatal(err)
	}
	nodeBytes := []byte("inert bundled Node fixture")
	if err := os.WriteFile(filepath.Join(base, "node", "node.exe"), nodeBytes, 0600); err != nil {
		t.Fatal(err)
	}

	writeFile := func(rel string, data []byte) {
		t.Helper()
		file := filepath.Join(vendor, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	digest := func(data []byte) string {
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	}
	bundle := []byte("// " + installedVisualReplyGateMarkers[0] + "\n// POST " + installedVisualReplyGateMarkers[1] + "\n")
	events, err := json.Marshal(desktopSessionEventTypes)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string]string{}
	for i, name := range sessionRuntimeArtifacts {
		var data []byte
		switch i {
		case 0:
			data = []byte(`{"upstream":"rc.2"}`)
		case 2:
			data = append(append([]byte("const KNOWN_SESSION_EVENT_TYPES = new Set("), events...), []byte(");")...)
		case 4:
			data = bundle
		default:
			data = []byte("// inert " + name)
		}
		writeFile(name, data)
		artifacts[name] = digest(data)
	}
	safety = sessionRuntimeSafety{
		ID:           previous.ID,
		BaseSHA256:   digest([]byte("base")),
		PatchSHA256:  digest([]byte("patch")),
		OutputSHA256: artifacts[sessionRuntimeArtifacts[4]],
	}
	rc2SessionRuntimeSafety = safety

	manifest = map[string]any{
		"schemaVersion": 1, "cliVersion": "0.2.1-rc.2",
		"harnessPin": map[string]string{
			"repo": "https://github.com/deepseek-ai/deepseek-harness.git",
			"ref":  "dsh-v0.2.1-rc.2", "sha": sessionRuntimeHarnessCommit, "npm": "0.2.1-rc.2",
		},
		"requiredEventTypes": desktopSessionEventTypes,
		"artifacts":          artifacts,
		"safetyPatch":        safety,
		"node":               map[string]string{"path": "node/node.exe", "sha256": digest(nodeBytes)},
	}
	writeManifest := func() {
		t.Helper()
		raw, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		writeFile(sessionRuntimeManifestName, raw)
	}
	writeManifest()
	raw, err := json.Marshal(safety)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(installedReadinessRecordName, raw)
	for _, name := range installedProjectArtifacts {
		writeFile(name, []byte("// inert "+name))
	}
	return vendor, safety, manifest
}

func TestInstalledRuntimeProofAcceptsTheStagedRc2(t *testing.T) {
	vendor, _, _ := installedProofFixture(t)
	// The source-build manifest must be absent for this path.
	if _, err := os.Stat(filepath.Join(vendor, "current-host-build.json")); !os.IsNotExist(err) {
		t.Fatal("fixture must not carry a source-build manifest")
	}
	c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
	if !c.ArtifactsReady || !c.ReadinessGuardReady || !c.VisualReplyReader {
		t.Fatalf("installed proof rejected: %+v", c)
	}
	if c.SourceCommit != sessionRuntimeHarnessCommit {
		t.Fatalf("source commit = %q, want the rc.2 pin", c.SourceCommit)
	}
	if c.Project || c.Team || c.VisualReplies || c.Reason == "" {
		t.Fatalf("bridges were not reported from the caller: %+v", c)
	}
	full := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{Project: true, ProjectOverlay: true, HTMLPreview: true, VisualReplyIsolation: true})
	if !full.Project || !full.Team || !full.VisualReplies || full.Reason != "" {
		t.Fatalf("full bridges on a proven runtime: %+v", full)
	}
	// The launch check still passes for the installed runtime, unchanged.
	if err := ValidateCurrentHostForLaunch(filepath.Join(vendor, "deepseek-harness")); err != nil {
		t.Fatalf("installed runtime rejected for launch: %v", err)
	}
}

func TestInstalledRuntimeProofRC4KeepsVersionAndGateTogether(t *testing.T) {
	vendor, safety, manifest := installedProofFixture(t)
	previous := rc4SessionRuntimeSafety
	rc4SessionRuntimeSafety = safety
	t.Cleanup(func() { rc4SessionRuntimeSafety = previous })
	manifest["cliVersion"] = "0.2.1-rc.4"
	pin := manifest["harnessPin"].(map[string]string)
	pin["ref"], pin["npm"], pin["sha"] = "dsh-v0.2.1-rc.4", "0.2.1-rc.4", sessionRuntimeRC4Commit
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
	c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
	if !c.ArtifactsReady || c.SourceCommit != sessionRuntimeRC4Commit {
		t.Fatalf("rc.4 capability identity: %+v", c)
	}
	if err := ValidateCurrentHostForLaunch(filepath.Join(vendor, "deepseek-harness")); err != nil {
		t.Fatal(err)
	}
	pin["sha"] = sessionRuntimeHarnessCommit
	write()
	if err := ValidateCurrentHostForLaunch(filepath.Join(vendor, "deepseek-harness")); err == nil {
		t.Fatal("rc.4 accepted an rc.2 provenance pin")
	}
	pin["sha"] = sessionRuntimeRC4Commit
	manifest["safetyPatch"] = previous
	write()
	if err := ValidateCurrentHostForLaunch(filepath.Join(vendor, "deepseek-harness")); err == nil {
		t.Fatal("rc.4 accepted a gate that differs from the bundled bytes")
	}
}

func TestInstalledRuntimeProofFailsClosed(t *testing.T) {
	t.Run("corrupted readiness record", func(t *testing.T) {
		vendor, safety, _ := installedProofFixture(t)
		safety.OutputSHA256 = strings.Repeat("0", 64)
		raw, _ := json.Marshal(safety)
		if err := os.WriteFile(filepath.Join(vendor, installedReadinessRecordName), raw, 0600); err != nil {
			t.Fatal(err)
		}
		c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
		if c.ArtifactsReady || !strings.Contains(c.Reason, "readiness record") {
			t.Fatalf("corrupted readiness record accepted: %+v", c)
		}
	})
	t.Run("missing readiness record", func(t *testing.T) {
		vendor, _, _ := installedProofFixture(t)
		if err := os.Remove(filepath.Join(vendor, installedReadinessRecordName)); err != nil {
			t.Fatal(err)
		}
		c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
		if c.ArtifactsReady || !strings.Contains(c.Reason, "readiness record is missing") {
			t.Fatalf("missing readiness record accepted: %+v", c)
		}
	})
	t.Run("missing gate marker", func(t *testing.T) {
		vendor, safety, manifest := installedProofFixture(t)
		// Keep the pin honest for the marker-less bytes, so only the marker check
		// can reject it.
		plain := []byte("// no gate here\n")
		bundle := filepath.Join(vendor, filepath.FromSlash(installedVisualRepliesBundle))
		if err := os.WriteFile(bundle, plain, 0600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(plain)
		digest := hex.EncodeToString(sum[:])
		safety.OutputSHA256 = digest
		rc2SessionRuntimeSafety = safety
		manifest["safetyPatch"] = safety
		manifest["artifacts"].(map[string]string)[installedVisualRepliesBundle] = digest
		raw, _ := json.Marshal(manifest)
		if err := os.WriteFile(filepath.Join(vendor, sessionRuntimeManifestName), raw, 0600); err != nil {
			t.Fatal(err)
		}
		record, _ := json.Marshal(safety)
		if err := os.WriteFile(filepath.Join(vendor, installedReadinessRecordName), record, 0600); err != nil {
			t.Fatal(err)
		}
		c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
		if c.ArtifactsReady || !strings.Contains(c.Reason, "lacks the readiness gate") {
			t.Fatalf("marker-less bundle accepted: %+v", c)
		}
	})
	t.Run("missing dsh-project package", func(t *testing.T) {
		vendor, _, _ := installedProofFixture(t)
		if err := os.Remove(filepath.Join(vendor, filepath.FromSlash(installedProjectArtifacts[3]))); err != nil {
			t.Fatal(err)
		}
		c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
		if c.ArtifactsReady || !strings.Contains(c.Reason, "Project package is incomplete") {
			t.Fatalf("incomplete Project package accepted: %+v", c)
		}
	})
	t.Run("no manifest at all", func(t *testing.T) {
		vendor := t.TempDir()
		c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
		if c.ArtifactsReady || c.Reason == "" {
			t.Fatalf("empty vendor accepted: %+v", c)
		}
	})
	t.Run("changed session artifact", func(t *testing.T) {
		vendor, _, _ := installedProofFixture(t)
		file := filepath.Join(vendor, filepath.FromSlash(sessionRuntimeArtifacts[1]))
		if err := os.WriteFile(file, []byte("// changed"), 0600); err != nil {
			t.Fatal(err)
		}
		c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
		if c.ArtifactsReady || !strings.Contains(c.Reason, "artifact changed") {
			t.Fatalf("changed artifact accepted: %+v", c)
		}
	})
}

// TestSourceBuildManifestKeepsItsBranch makes sure an installed proof never
// speaks for a source build: when current-host-build.json is present, only that
// manifest decides, exactly as before.
func TestSourceBuildManifestKeepsItsBranch(t *testing.T) {
	vendor, _, _ := installedProofFixture(t)
	raw, err := json.Marshal(map[string]any{
		"schemaVersion": 1, "sourceCommit": "unsupported", "buildProfile": "official",
		"platform": "win32", "arch": "x64", "artifacts": map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "current-host-build.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	c := ReadCurrentHostCapabilities(vendor, CurrentHostBridges{})
	if c.ArtifactsReady || !strings.Contains(c.Reason, "provenance is unsupported") {
		t.Fatalf("installed proof overrode the source-build manifest: %+v", c)
	}
}
