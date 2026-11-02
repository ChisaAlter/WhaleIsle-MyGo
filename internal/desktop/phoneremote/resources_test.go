package phoneremote

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPackagedMobileBuiltFixture(t *testing.T) {
	vendor := os.Getenv("WHALEISLE_MOBILE_RELEASE_FIXTURE_VENDOR")
	if vendor == "" {
		t.Skip("set WHALEISLE_MOBILE_RELEASE_FIXTURE_VENDOR to a real closed release stage")
	}
	r, err := ResolveResources(t.TempDir(), vendor)
	if err != nil {
		t.Fatal(err)
	}
	if r.Node != filepath.Join(vendor, "dshd-remote", "node.exe") {
		t.Fatalf("unexpected recorded mobile executable: %s", r.Node)
	}
}

func TestPackagedMobileResourceProof(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vendor", "dshd-remote")
	write := func(relative string, value any) {
		t.Helper()
		filename := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		var bytes []byte
		switch value := value.(type) {
		case string:
			bytes = []byte(value)
		default:
			var err error
			bytes, err = json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filename, bytes, 0600); err != nil {
			t.Fatal(err)
		}
	}
	digest := func(value string) string { sum := sha256.Sum256([]byte(value)); return hex.EncodeToString(sum[:]) }
	platform, arch := runtime.GOOS, runtime.GOARCH
	if platform == "windows" {
		platform = "win32"
	}
	if arch == "amd64" {
		arch = "x64"
	}
	write("node.exe", "inert node")
	sqlite := "node_modules/better-sqlite3/build/Release/better_sqlite3.node"
	pty := "node_modules/node-pty/prebuilds/" + platform + "-" + arch + "/conpty.node"
	write(sqlite, "inert sqlite")
	write(pty, "inert pty")
	write("node-build.json", map[string]any{"platform": platform, "arch": arch, "runtime": "node", "abi": "137", "nodeSHA256": digest("inert node"), "nativeArtifacts": map[string]string{sqlite: digest("inert sqlite"), pty: digest("inert pty")}})
	files, links, err := mobileGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	m := mobileResourceManifest{Format: 3, Profile: "installed-rc2", Platform: platform, Arch: arch, NodePath: "node.exe", Files: files, Links: links}
	m.HarnessPeer.Path = "../deepseek-harness"
	m.HarnessPeer.Artifacts = map[string]string{}
	for _, name := range []string{"apps/cli/lib/bin.js", "packages/core/session/lib/index.js", "packages/session/session-persistence/lib/index.js"} {
		filename := filepath.Join(filepath.Dir(root), "deepseek-harness", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		m.HarnessPeer.Artifacts[name] = digest(name)
	}
	write("staging-manifest.json", m)
	node, err := verifyPackagedMobileResources(root)
	if err != nil || node != filepath.Join(root, "node.exe") {
		t.Fatalf("valid proof failed: %q %v", node, err)
	}
	write("unknown.js", "unexpected executable code")
	if _, err := verifyPackagedMobileResources(root); err == nil {
		t.Fatal("extra untracked file accepted")
	}
	if err := os.Remove(filepath.Join(root, "unknown.js")); err != nil {
		t.Fatal(err)
	}
	write(sqlite, "changed sqlite")
	if _, err := verifyPackagedMobileResources(root); err == nil {
		t.Fatal("changed native binding accepted")
	}
}
