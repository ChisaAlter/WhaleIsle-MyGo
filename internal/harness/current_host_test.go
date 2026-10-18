package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Files here are inert fixture bytes, not an executable Host assembly.
func TestCurrentHostPinnedAssemblies(t *testing.T) {
	for _, commit := range []string{CurrentHostSourceCommit, CurrentHostE9SourceCommit} {
		t.Run(commit[:8], func(t *testing.T) {
			root := t.TempDir()
			hashes := map[string]string{}
			for _, name := range currentHostArtifacts {
				file := filepath.Join(root, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(name), 0600); err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256([]byte(name))
				hashes[name] = hex.EncodeToString(sum[:])
			}
			platform, arch := runtime.GOOS, runtime.GOARCH
			if platform == "windows" {
				platform = "win32"
			}
			if arch == "amd64" {
				arch = "x64"
			}
			safety := map[string]string{"id": CurrentHostReadinessGuardID, "patchSHA256": CurrentHostReadinessPatchSHA256, "sourceSHA256": "8c73f32eb14e9ba46d3712bcbc569b2effe74c28bc60fbfa299e08c920bf2812"}
			m := map[string]any{"schemaVersion": 1, "sourceCommit": commit, "buildProfile": "official", "platform": platform, "arch": arch, "artifacts": hashes, "safetyPatch": safety}
			write := func() {
				t.Helper()
				raw, err := json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(root, "current-host-build.json"), raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			write()
			c := ReadCurrentHostCapabilities(root, CurrentHostBridges{})
			if !c.ArtifactsReady || !c.ReadinessGuardReady || c.SourceCommit != commit || c.Project || c.Team || c.VisualReplies {
				t.Fatalf("unexpected capability defaults: %+v", c)
			}
			if err := ValidateCurrentHostForLaunch(filepath.Join(root, "deepseek-harness")); err != nil {
				t.Fatal(err)
			}
			safety["sourceSHA256"] = "unpatched"
			write()
			if err := ValidateCurrentHostForLaunch(filepath.Join(root, "deepseek-harness")); err == nil {
				t.Fatal("missing safety patch accepted")
			}
			m["sourceCommit"] = "unsupported"
			write()
			if ReadCurrentHostCapabilities(root, CurrentHostBridges{}).ArtifactsReady {
				t.Fatal("unknown source accepted")
			}
			m["sourceCommit"] = commit
			m["arch"] = "other"
			write()
			if ReadCurrentHostCapabilities(root, CurrentHostBridges{}).ArtifactsReady {
				t.Fatal("wrong architecture accepted")
			}
			m["arch"] = arch
			write()
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(currentHostArtifacts[0])), []byte("changed"), 0600); err != nil {
				t.Fatal(err)
			}
			if ReadCurrentHostCapabilities(root, CurrentHostBridges{}).ArtifactsReady {
				t.Fatal("changed artifact accepted")
			}
		})
	}
}
