package launcherctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

var (
	portableRootMu sync.RWMutex
	portableRoot   string
)

// PortableDesktopRoot returns the verified adjacent portable bundle root, if active.
func PortableDesktopRoot() string {
	portableRootMu.RLock()
	defer portableRootMu.RUnlock()
	return portableRoot
}

// SetPortableDesktopRoot swaps the stored portable root (primarily for tests) and returns the previous value.
func SetPortableDesktopRoot(root string) string {
	portableRootMu.Lock()
	prev := portableRoot
	portableRoot = root
	portableRootMu.Unlock()
	return prev
}

// PortableDesktopManifest defines the fixed contract inside portable-package.json.
type PortableDesktopManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	UserData      string `json:"userData"`
	Runtime       string `json:"runtime"`
	Entry         string `json:"entry"`
}

// PreparePortableDesktop checks the running executable's directory for a portable package manifest.
func PreparePortableDesktop() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return PreparePortableDesktopAt(executable)
}

// PreparePortableDesktopAt validates and configures portable execution rooted beside executable.
func PreparePortableDesktopAt(executable string) error {
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
		return nil
	}
	if err != nil {
		return fmt.Errorf("portable package: %w", err)
	}
	if err := PortableContainedPath(root, manifestPath, false); err != nil {
		return err
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("portable package: %w", err)
	}
	var manifest PortableDesktopManifest
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
	pnpm := filepath.Join(root, "runtime", "pnpm", "pnpm.exe")
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
	for _, path := range []string{manifestPath, filepath.Join(root, "runtime"), vendor, node, pnpm, cli} {
		if err := PortableContainedPath(root, path, false); err != nil {
			return err
		}
	}
	for _, relative := range []string{"", config.FileName, "credentials.json", "dsh-home", "runtime", "native", "components"} {
		if err := PortableContainedPath(userData, filepath.Join(userData, relative), true); err != nil {
			return err
		}
	}
	if err := PortableContainedPath(root, userData, true); err != nil {
		return err
	}
	for _, artifact := range []string{
		"host-session-runtime.json", "harness-upstream.json",
		"deepseek-harness/packages/core/session/lib/index.js",
		"deepseek-harness/packages/session/session-persistence/lib/index.js",
		"deepseek-harness/packages/client/ui-visual-replies/lib/index.js",
		"deepseek-harness/apps/web/dist/index.html",
	} {
		if err := PortableContainedPath(vendor, filepath.Join(vendor, filepath.FromSlash(artifact)), false); err != nil {
			return fmt.Errorf("portable package requires a private verified session runtime: %w", err)
		}
	}
	if _, err := harness.Resolve(harness.LaunchOptions{Host: "127.0.0.1", VendorDir: vendor, HarnessRoot: filepath.Join(vendor, "deepseek-harness"), NodeBin: node}, 3080); err != nil {
		return fmt.Errorf("portable runtime: %w", err)
	}
	if stat, err := os.Stat(pnpm); err != nil || !stat.Mode().IsRegular() {
		return fmt.Errorf("portable package requires its bundled native pnpm executable")
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
		if err := PortableContainedPath(userData, workspace, true); err != nil {
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
	if err := preparePortablePackageTools(root, userData); err != nil {
		return err
	}
	if err := os.Setenv(paths.EnvUserData, userData); err != nil {
		return err
	}
	if err := os.Setenv(paths.EnvVendor, vendor); err != nil {
		return err
	}
	SetPortableDesktopRoot(root)
	return nil
}

func preparePortablePackageTools(root, userData string) error {
	manager := filepath.Join(userData, "package-manager")
	if err := PortableContainedPath(userData, manager, true); err != nil {
		return err
	}
	if err := os.MkdirAll(manager, 0o700); err != nil {
		return err
	}
	for variable, name := range map[string]string{"NPM_CONFIG_USERCONFIG": "user.npmrc", "NPM_CONFIG_GLOBALCONFIG": "global.npmrc"} {
		filename := filepath.Join(manager, name)
		if err := PortableContainedPath(userData, filename, true); err != nil {
			return err
		}
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			err = file.Close()
		} else if os.IsExist(err) {
			err = nil
		}
		if err != nil {
			return err
		}
		if err := os.Setenv(variable, filename); err != nil {
			return err
		}
	}
	if err := os.Setenv("NPM_CONFIG_STORE_DIR", filepath.Join(manager, "store")); err != nil {
		return err
	}
	prefix := filepath.Join(root, "runtime", "node") + string(os.PathListSeparator) + filepath.Join(root, "runtime", "pnpm")
	return os.Setenv("PATH", prefix+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func samePortablePath(a, b string) bool {
	return filepath.IsAbs(a) && filepath.IsAbs(b) && strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// PortableContainedPath ensures an existing or missing child resolves strictly inside root.
func PortableContainedPath(root, path string, missing bool) error {
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
		if samePortablePath(probe, root) {
			resolvedParent, err := filepath.EvalSymlinks(parent)
			if err == nil && samePortablePath(resolvedParent, parent) {
				return nil
			}
		}
		probe = parent
	}
}
