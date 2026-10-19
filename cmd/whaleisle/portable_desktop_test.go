package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/paths"
)

func portableTestEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv(paths.EnvUserData, "")
	t.Setenv(paths.EnvVendor, "")
	t.Setenv("WHALEISLE_ISOLATED", "")
	previous := portableDesktopRoot
	portableDesktopRoot = ""
	t.Cleanup(func() { portableDesktopRoot = previous })
}

func writePortableManifest(t *testing.T, root string, manifest portableDesktopManifest) {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "portable-package.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func validPortableManifest() portableDesktopManifest {
	return portableDesktopManifest{1, "whaleisle-isolated-portable", "data", "runtime", "WhaleIsle.exe"}
}

func TestPortableMissingManifestLeavesInstalledStartupAlone(t *testing.T) {
	portableTestEnvironment(t)
	data, vendor := t.TempDir(), t.TempDir()
	t.Setenv(paths.EnvUserData, data)
	t.Setenv(paths.EnvVendor, vendor)
	root := t.TempDir()
	if err := preparePortableDesktopAt(filepath.Join(root, "whaleisle.exe")); err != nil {
		t.Fatal(err)
	}
	if isolatedDesktopRun() || os.Getenv(paths.EnvUserData) != data || os.Getenv(paths.EnvVendor) != vendor {
		t.Fatal("ordinary startup was redirected without a package manifest")
	}
}

func TestPortableRejectsOtherManifestPathsAndEntryBeforeInitialization(t *testing.T) {
	for _, field := range []string{"schema", "kind", "data", "runtime", "entry", "renamed-executable"} {
		t.Run(field, func(t *testing.T) {
			portableTestEnvironment(t)
			root := t.TempDir()
			manifest := validPortableManifest()
			entry := "WhaleIsle.exe"
			switch field {
			case "schema":
				manifest.SchemaVersion = 2
			case "kind":
				manifest.Kind = "installed"
			case "data":
				manifest.UserData = "../outside"
			case "runtime":
				manifest.Runtime = t.TempDir()
			case "entry":
				manifest.Entry = "other.exe"
			case "renamed-executable":
				entry = "other.exe"
			}
			writePortableManifest(t, root, manifest)
			if err := preparePortableDesktopAt(filepath.Join(root, entry)); err == nil {
				t.Fatal("nonfixed package contract was accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) || portableDesktopRoot != "" {
				t.Fatal("invalid package initialized private data or activated portable mode")
			}
		})
	}
}

func TestPortableRejectsExternalOverridesAndUnverifiedRuntime(t *testing.T) {
	for _, key := range []string{paths.EnvUserData, paths.EnvVendor, "missing-runtime"} {
		t.Run(key, func(t *testing.T) {
			portableTestEnvironment(t)
			root := t.TempDir()
			writePortableManifest(t, root, validPortableManifest())
			if key != "missing-runtime" {
				t.Setenv(key, t.TempDir())
			}
			if err := preparePortableDesktopAt(filepath.Join(root, "WhaleIsle.exe")); err == nil {
				t.Fatal("external override or missing runtime was accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
				t.Fatal("failed validation initialized data")
			}
		})
	}
}

func TestPortablePrivatePathBoundsAndFreshChildren(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "data")
	if err := portableContainedPath(private, filepath.Join(private, "config.json"), true); err != nil {
		t.Fatalf("fresh private child rejected: %v", err)
	}
	if err := portableContainedPath(private, root, false); err == nil {
		t.Fatal("existing path outside the private root accepted")
	}
}

// This opt-in integration reads only the already staged public runtime. It
// copies the exact gate inputs, never starts Node/Host or copies user profiles.
func TestPortableVerifiedRuntimeInitializesAndPreservesPrivateConfig(t *testing.T) {
	portableTestEnvironment(t)
	source := os.Getenv("WHALEISLE_PORTABLE_TEST_RUNTIME")
	if source == "" {
		t.Skip("explicit verified runtime fixture directory required")
	}
	root := t.TempDir()
	for _, relative := range []string{
		"node/node.exe", "vendor/host-session-runtime.json", "vendor/harness-upstream.json",
		"vendor/deepseek-harness/apps/cli/lib/bin.js", "vendor/deepseek-harness/apps/web/dist/index.html",
		"vendor/deepseek-harness/packages/core/session/lib/index.js",
		"vendor/deepseek-harness/packages/session/session-persistence/lib/index.js",
		"vendor/deepseek-harness/packages/client/ui-visual-replies/lib/index.js",
	} {
		copyPortableGateInput(t, filepath.Join(source, filepath.FromSlash(relative)), filepath.Join(root, "runtime", filepath.FromSlash(relative)))
	}
	writePortableManifest(t, root, validPortableManifest())
	entry := filepath.Join(root, "WhaleIsle.exe")
	if err := preparePortableDesktopAt(entry); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, "data")
	if paths.UserDataDir() != data || paths.VendorDir() != filepath.Join(root, "runtime", "vendor") || !isolatedDesktopRun() || isolatedDesktopHidden() {
		t.Fatal("portable startup did not choose private visible mode")
	}
	store, err := config.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	cfg := store.Get()
	if cfg.Workspace != filepath.Join(data, "workspace") || cfg.AutoStartDesktop != config.Defaults().AutoStartDesktop || cfg.Theme != config.Defaults().Theme {
		t.Fatal("first run did not preserve the original desktop defaults and private workspace")
	}
	if _, err := os.Stat(cfg.Workspace); err != nil {
		t.Fatal(err)
	}
	retained := []byte("{\n  \"locale\": \"en\", \"autoStartDesktop\": false, \"portableUnknown\": {\"keep\":true}\n}\n")
	if err := os.WriteFile(store.Path(), retained, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := preparePortableDesktopAt(entry); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(store.Path())
	if err != nil || !bytes.Equal(got, retained) {
		t.Fatal("existing portable configuration was rewritten")
	}
	if err := os.WriteFile(store.Path(), []byte(`{"nodeBin":"C:\\outside\\node.exe"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := preparePortableDesktopAt(entry); err == nil {
		t.Fatal("existing config selected an external Node")
	}
}

func copyPortableGateInput(t *testing.T, source, target string) {
	t.Helper()
	in, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatalf("copy public runtime gate input: %v %v", copyErr, closeErr)
	}
}
