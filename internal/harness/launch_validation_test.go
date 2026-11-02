package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"whaleisle/internal/paths"
)

func launchValidationRegistry(root string) string {
	return filepath.Join(root, "packages", "core", "session", "lib", "index.js")
}

func launchValidationWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func launchValidationManifest(t *testing.T, vendor string, manifest map[string]any) {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	launchValidationWrite(t, filepath.Join(vendor, sessionRuntimeManifestName), data)
}

// Count only actual vocabulary reads; all bytes still come from the fixture
// file, and no production globals or content/hash validation are replaced.
func launchValidationReader(counts map[string]int) sessionRuntimeRegistryReader {
	return func(path string) ([]byte, error) {
		counts[filepath.Clean(path)]++
		return os.ReadFile(path)
	}
}

func launchValidationInstalled(t *testing.T) (LaunchOptions, map[string]any) {
	t.Helper()
	vendor, _, manifest := installedProofFixture(t)
	root := filepath.Join(vendor, "deepseek-harness")
	launchValidationWrite(t, filepath.Join(root, "apps", "web", "dist", "index.html"), []byte("inert web fixture"))
	return LaunchOptions{HarnessRoot: root, VendorDir: vendor, NodeBin: filepath.Join(filepath.Dir(vendor), "node", "node.exe"), Home: filepath.Join(t.TempDir(), "home"), Host: "127.0.0.1"}, manifest
}

func launchValidationSource(t *testing.T, current bool) LaunchOptions {
	t.Helper()
	vendor := t.TempDir()
	root := filepath.Join(vendor, "deepseek-harness")
	if current {
		hashes := map[string]string{}
		for _, name := range currentHostArtifacts {
			data := []byte("inert source artifact " + name)
			launchValidationWrite(t, filepath.Join(vendor, filepath.FromSlash(name)), data)
			sum := sha256.Sum256(data)
			hashes[name] = hex.EncodeToString(sum[:])
		}
		platform, arch := runtime.GOOS, runtime.GOARCH
		if platform == "windows" {
			platform = "win32"
		}
		if arch == "amd64" {
			arch = "x64"
		}
		manifest := map[string]any{"schemaVersion": 1, "sourceCommit": CurrentHostSourceCommit, "buildProfile": "official", "platform": platform, "arch": arch, "artifacts": hashes, "safetyPatch": map[string]string{"id": CurrentHostReadinessGuardID, "patchSHA256": CurrentHostReadinessPatchSHA256, "sourceSHA256": "8c73f32eb14e9ba46d3712bcbc569b2effe74c28bc60fbfa299e08c920bf2812"}}
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		launchValidationWrite(t, filepath.Join(vendor, "current-host-build.json"), data)
	} else {
		launchValidationWrite(t, filepath.Join(root, "apps", "cli", "lib", "bin.js"), []byte("inert legacy cli"))
		launchValidationWrite(t, filepath.Join(root, "apps", "web", "dist", "index.html"), []byte("inert legacy web"))
	}
	writeSessionRuntimeRegistry(t, root, desktopSessionEventTypes)
	node := filepath.Join(t.TempDir(), "node.exe")
	launchValidationWrite(t, node, []byte("inert source Node"))
	return LaunchOptions{HarnessRoot: root, VendorDir: vendor, NodeBin: node, Host: "127.0.0.1"}
}

func TestLaunchValidationReceiptInstalledReadsOncePerResolve(t *testing.T) {
	for _, selection := range []string{"root", "vendor", "configured"} {
		t.Run(selection, func(t *testing.T) {
			o, _ := launchValidationInstalled(t)
			root := o.HarnessRoot
			switch selection {
			case "vendor":
				o.HarnessRoot = ""
			case "configured":
				o.DshBin = filepath.Join(root, "apps", "cli", "lib", "bin.js")
			}
			counts := map[string]int{}
			read := launchValidationReader(counts)
			for pass := 1; pass <= 2; pass++ {
				plan, err := resolveWithRegistryReader(o, 3080, read)
				if err != nil {
					t.Fatal(err)
				}
				if plan.Command != o.NodeBin {
					t.Fatalf("wrong selected Node: %+v", plan)
				}
				if counts[launchValidationRegistry(root)] != pass {
					t.Fatalf("Resolve pass%d registry reads=%v, want one fresh read per pass", pass, counts)
				}
			}
			if len(counts) != 1 {
				t.Fatalf("unexpected registry root: %v", counts)
			}
		})
	}
}

func TestLaunchValidationReceiptSourceAndLegacyKeepVocabularyChecks(t *testing.T) {
	for _, current := range []bool{false, true} {
		for _, configured := range []bool{false, true} {
			name := "legacy"
			if current {
				name = "source"
			}
			if configured {
				name += "_configured"
			}
			t.Run(name, func(t *testing.T) {
				o := launchValidationSource(t, current)
				root := o.HarnessRoot
				if configured {
					o.DshBin = filepath.Join(root, "apps", "cli", "lib", "bin.js")
				}
				counts := map[string]int{}
				read := launchValidationReader(counts)
				if _, err := resolveWithRegistryReader(o, 3080, read); err != nil {
					t.Fatal(err)
				}
				if counts[launchValidationRegistry(root)] != 1 {
					t.Fatalf("source registry reads=%v", counts)
				}
				writeSessionRuntimeRegistry(t, root, desktopSessionEventTypes[:2])
				if _, err := resolveWithRegistryReader(o, 3080, read); err == nil || !strings.Contains(err.Error(), "older than") {
					t.Fatalf("stale source registry accepted: %v", err)
				}
				if counts[launchValidationRegistry(root)] != 2 {
					t.Fatalf("changed source was not reread: %v", counts)
				}
			})
		}
	}
}

func TestLaunchValidationReceiptIsRootBound(t *testing.T) {
	o, _ := launchValidationInstalled(t)
	canonical := o.HarnessRoot
	other := filepath.Join(o.VendorDir, "other-harness")
	launchValidationWrite(t, filepath.Join(other, "apps", "cli", "lib", "bin.js"), []byte("inert other cli"))
	launchValidationWrite(t, filepath.Join(other, "apps", "web", "dist", "index.html"), []byte("inert other web"))
	writeSessionRuntimeRegistry(t, other, desktopSessionEventTypes[:2])
	o.HarnessRoot = other
	counts := map[string]int{}
	_, err := resolveWithRegistryReader(o, 3080, launchValidationReader(counts))
	if err == nil || !strings.Contains(err.Error(), "older than") {
		t.Fatalf("canonical receipt certified a different root: %v", err)
	}
	if counts[launchValidationRegistry(canonical)] != 1 || counts[launchValidationRegistry(other)] != 1 {
		t.Fatalf("different-root reads=%v", counts)
	}
}

func TestLaunchValidationReceiptIsNeverIssuedForFailedInstalledProof(t *testing.T) {
	for _, test := range []struct {
		name, want string
		edit       func(*testing.T, LaunchOptions, map[string]any)
	}{
		{"provenance", "unsupported provenance", func(t *testing.T, o LaunchOptions, m map[string]any) {
			m["schemaVersion"] = 2
			launchValidationManifest(t, o.VendorDir, m)
		}},
		{"node", "selected Node differs", func(t *testing.T, o LaunchOptions, _ map[string]any) {
			launchValidationWrite(t, o.NodeBin, []byte("changed and longer Node fixture"))
		}},
		{"artifact", "artifact changed", func(t *testing.T, o LaunchOptions, _ map[string]any) {
			launchValidationWrite(t, launchValidationRegistry(o.HarnessRoot), []byte("changed registry bytes"))
		}},
		{"visual-pin", "visual-readiness", func(t *testing.T, o LaunchOptions, m map[string]any) {
			m["safetyPatch"] = sessionRuntimeSafety{ID: rc2SessionRuntimeSafety.ID}
			launchValidationManifest(t, o.VendorDir, m)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			o, m := launchValidationInstalled(t)
			test.edit(t, o, m)
			counts := map[string]int{}
			read := launchValidationReader(counts)
			receipt, err := validateCurrentHostForLaunch(o.HarnessRoot, read)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("failed proof accepted: %v", err)
			}
			if receipt.sessionRoot != "" {
				t.Fatal("failed proof issued receipt")
			}
			if len(counts) != 0 {
				t.Fatalf("registry read before provenance/artifact/Node proof: %v", counts)
			}
			if _, err := resolveWithRegistryReader(o, 3080, read); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Resolve weakened proof error: %v", err)
			}
		})
	}
}

func TestLaunchValidationReceiptRegistryErrorsStayFailClosed(t *testing.T) {
	for _, current := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "installed"}[current], func(t *testing.T) {
			var o LaunchOptions
			if current {
				o, _ = launchValidationInstalled(t)
			} else {
				o = launchValidationSource(t, false)
			}
			want := errors.New("fixture reader permission denied")
			calls := 0
			_, err := resolveWithRegistryReader(o, 3080, func(path string) ([]byte, error) { calls++; return nil, want })
			if calls != 1 || !errors.Is(err, want) || !strings.Contains(err.Error(), "Host session reader unavailable") {
				t.Fatalf("registry error was suppressed/retried: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestLaunchValidationReceiptStillChecksExplicitNodeAfterReceipt(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "vendor"
		if configured {
			name = "configured"
		}
		t.Run(name, func(t *testing.T) {
			o, _ := launchValidationInstalled(t)
			bundled, err := os.ReadFile(o.NodeBin)
			if err != nil {
				t.Fatal(err)
			}
			o.NodeBin = filepath.Join(t.TempDir(), "explicit-node.exe")
			launchValidationWrite(t, o.NodeBin, bundled)
			if configured {
				o.DshBin = filepath.Join(o.HarnessRoot, "apps", "cli", "lib", "bin.js")
			}
			counts := map[string]int{}
			read := launchValidationReader(counts)
			if _, err := resolveWithRegistryReader(o, 3080, read); err != nil {
				t.Fatal(err)
			}
			launchValidationWrite(t, o.NodeBin, []byte("explicit Node changed to different longer bytes"))
			if _, err := resolveWithRegistryReader(o, 3080, read); err == nil || !strings.Contains(err.Error(), "selected Node differs") {
				t.Fatalf("receipt waived the selected Node identity: %v", err)
			}
			if counts[launchValidationRegistry(o.HarnessRoot)] != 2 {
				t.Fatalf("fresh installed proof reads=%v", counts)
			}
		})
	}
}

func TestLaunchValidationReceiptPreservesCustomScriptResolution(t *testing.T) {
	o, _ := launchValidationInstalled(t)
	// A manually configured custom script has no inferred standard CLI root;
	// retain its existing behavior instead of applying an unrelated receipt.
	o.DshBin = filepath.Join(t.TempDir(), "custom-launch.mjs")
	launchValidationWrite(t, o.DshBin, []byte("inert custom script"))
	counts := map[string]int{}
	plan, err := resolveWithRegistryReader(o, 3080, launchValidationReader(counts))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != "configured" || plan.Command != o.NodeBin || len(plan.Args) == 0 || plan.Args[0] != o.DshBin || len(counts) != 0 {
		t.Fatalf("custom script resolution changed: plan=%+v reads=%v", plan, counts)
	}
}

func TestLaunchValidationReceiptPrepareKeepsTwoIndependentPasses(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		name := "unchanged"
		if corrupt {
			name = "prepared_registry_changed"
		}
		t.Run(name, func(t *testing.T) {
			o, manifest := launchValidationInstalled(t)
			userData := t.TempDir()
			t.Setenv(paths.EnvUserData, userData)
			o.Home = paths.DshHome()
			profile := paths.WebProfileDir()
			for _, plugin := range builtinPlugins() {
				raw, _ := json.Marshal(map[string]string{"name": plugin.Package})
				launchValidationWrite(t, filepath.Join(o.VendorDir, plugin.VendorName, "package.json"), raw)
			}
			counts := map[string]int{}
			read := launchValidationReader(counts)
			if _, err := resolveWithRegistryReader(o, 3080, read); err != nil {
				t.Fatal(err)
			}
			if counts[launchValidationRegistry(o.HarnessRoot)] != 1 {
				t.Fatalf("pre-Prepare reads=%v", counts)
			}
			if err := paths.EnsureDshHome(); err != nil {
				t.Fatal(err)
			}
			patches, err := Prepare(PrepareOptions{VendorDir: o.VendorDir, ProfileDir: profile, DshHome: o.Home, HarnessRoot: o.HarnessRoot, RemoteSet: true, Remote: false})
			if err != nil {
				t.Fatal(err)
			}
			if len(patches) == 0 {
				t.Fatal("actual Prepare produced no fixture overlays")
			}
			o.PatchFiles = append(o.PatchFiles, patches...)
			if corrupt {
				// Re-pin the modified artifact as well: the final vocabulary read,
				// not only an earlier digest mismatch, must reject the new reader.
				writeSessionRuntimeRegistry(t, o.HarnessRoot, desktopSessionEventTypes[:2])
				data, err := os.ReadFile(launchValidationRegistry(o.HarnessRoot))
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(data)
				manifest["artifacts"].(map[string]string)[sessionRuntimeArtifacts[2]] = hex.EncodeToString(sum[:])
				launchValidationManifest(t, o.VendorDir, manifest)
			}
			_, err = resolveWithRegistryReader(o, 3080, read)
			if corrupt {
				if err == nil || !strings.Contains(err.Error(), "older than") {
					t.Fatalf("post-Prepare stale registry accepted: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if counts[launchValidationRegistry(o.HarnessRoot)] != 2 {
				t.Fatalf("final Resolve did not reread registry: %v", counts)
			}
		})
	}
}
