package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// Set only after the adjacent package and its private runtime pass validation.
// An environment variable alone cannot turn an installed desktop into portable.
var portableDesktopRoot string

type portableDesktopManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	UserData      string `json:"userData"`
	Runtime       string `json:"runtime"`
	Entry         string `json:"entry"`
}

func preparePortableDesktop() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return preparePortableDesktopAt(executable)
}

func preparePortableDesktopAt(executable string) error {
	root, err := filepath.EvalSymlinks(filepath.Dir(executable))
	if err != nil {
		return err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(root, "portable-package.json")
	_, err = os.Lstat(manifestPath)
	if os.IsNotExist(err) {
		return nil // Ordinary installed startup retains its existing behavior.
	}
	if err != nil {
		return fmt.Errorf("portable package: %w", err)
	}
	if err := portableContainedPath(root, manifestPath, false); err != nil {
		return err
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("portable package: %w", err)
	}
	var manifest portableDesktopManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("portable package: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Kind != "whaleisle-isolated-portable" ||
		manifest.UserData != "data" || manifest.Runtime != "runtime" || manifest.Entry != "WhaleIsle.exe" ||
		!strings.EqualFold(filepath.Base(executable), manifest.Entry) {
		return fmt.Errorf("portable package must use schema 1, WhaleIsle.exe, data and runtime")
	}
	userData := filepath.Join(root, "data")
	vendor := filepath.Join(root, "runtime", "vendor")
	node := filepath.Join(root, "runtime", "node", "node.exe")
	cli := filepath.Join(vendor, "deepseek-harness", "apps", "cli", "lib", "bin.js")
	for key, want := range map[string]string{paths.EnvUserData: userData, paths.EnvVendor: vendor} {
		if override := os.Getenv(key); override != "" && !samePortablePath(override, want) {
			return fmt.Errorf("portable package refuses external %s; use its adjacent %s", key, filepath.Base(want))
		}
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	if samePortablePath(userData, filepath.Join(base, config.DataDirName)) {
		return fmt.Errorf("portable package cannot use the installed application's data directory")
	}
	// Internal runtime dependency junctions remain valid. Only the runtime and
	// data roots, and the files we use, must resolve inside this package.
	for _, path := range []string{manifestPath, filepath.Join(root, "runtime"), vendor, node, cli} {
		if err := portableContainedPath(root, path, false); err != nil {
			return err
		}
	}
	for _, relative := range []string{"", config.FileName, "credentials.json", "dsh-home", "runtime", "native", "components"} {
		if err := portableContainedPath(userData, filepath.Join(userData, relative), true); err != nil {
			return err
		}
	}
	if err := portableContainedPath(root, userData, true); err != nil {
		return err
	}
	for _, artifact := range []string{
		"host-session-runtime.json", "harness-upstream.json",
		"deepseek-harness/packages/core/session/lib/index.js",
		"deepseek-harness/packages/session/session-persistence/lib/index.js",
		"deepseek-harness/packages/client/ui-visual-replies/lib/index.js",
		"deepseek-harness/apps/web/dist/index.html",
	} {
		if err := portableContainedPath(vendor, filepath.Join(vendor, filepath.FromSlash(artifact)), false); err != nil {
			return fmt.Errorf("portable package requires a private verified session runtime: %w", err)
		}
	}
	if _, err := harness.Resolve(harness.LaunchOptions{Host: "127.0.0.1", VendorDir: vendor, HarnessRoot: filepath.Join(vendor, "deepseek-harness"), NodeBin: node}, 3080); err != nil {
		return fmt.Errorf("portable runtime: %w", err)
	}
	store, err := config.Open(userData)
	if err != nil {
		return err
	}
	cfg := store.Get()
	for _, choice := range []struct{ name, path, want string }{{"nodeBin", cfg.NodeBin, node}, {"dshBin", cfg.DshBin, cli}} {
		if choice.path != "" && !samePortablePath(choice.path, choice.want) {
			return fmt.Errorf("portable config refuses external %s; clear the override in its private config", choice.name)
		}
	}
	if _, err := os.Stat(store.Path()); os.IsNotExist(err) {
		workspace := filepath.Join(userData, "workspace")
		if err := portableContainedPath(userData, workspace, true); err != nil {
			return err
		}
		if err := os.MkdirAll(workspace, 0o700); err != nil {
			return err
		}
		if err := store.Update(func(c *config.Config) { *c = config.Defaults(); c.Workspace = workspace }); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := os.Setenv(paths.EnvUserData, userData); err != nil {
		return err
	}
	if err := os.Setenv(paths.EnvVendor, vendor); err != nil {
		return err
	}
	portableDesktopRoot = root
	return nil
}

func samePortablePath(a, b string) bool {
	return filepath.IsAbs(a) && filepath.IsAbs(b) && strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// A missing private path is checked through its closest existing parent before
// creation. Existing junctions may resolve only within the named private root.
func portableContainedPath(root, path string, missing bool) error {
	probe := filepath.Clean(path)
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			rel, err := filepath.Rel(root, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
				return fmt.Errorf("portable path escapes its private root: %s", path)
			}
			return nil
		}
		if !missing || !os.IsNotExist(err) {
			return fmt.Errorf("portable path %s: %w", path, err)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return fmt.Errorf("portable path has no existing private parent: %s", path)
		}
		// For a fresh data root its existing package parent is legitimate;
		// subsequent creation is still at the original fixed data path.
		if samePortablePath(probe, root) {
			resolvedParent, err := filepath.EvalSymlinks(parent)
			if err == nil && samePortablePath(resolvedParent, parent) {
				return nil
			}
		}
		probe = parent
	}
}
