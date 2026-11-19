// Package paths resolves the file system locations MyGo鲸屿 uses:
// the user data directory (its own by default, or wherever the user recorded
// through the data-location record), the dsh home inside it, and the runtime
// directory holding the pid file.
//
// Path lookup may read a managed runtime selection or probe candidate files;
// it never starts a runtime or changes the selection. Tests override user data with
// WHALEISLE_USER_DATA (see UserDataDir).
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// EnvUserData, when set, overrides the user data directory. Tests use it
// to point the paths at a temporary directory.
const EnvUserData = "WHALEISLE_USER_DATA"

// UserDataDir returns the user data directory: the WHALEISLE_USER_DATA
// override when set, else the location recorded by the user, else this
// build's default (%AppData%\WhaleIsle-MyGo\data on Windows). A record that
// cannot be read, or that names a directory which is no longer there, falls
// back to the default and logs why. It does not create the directory; see
// EnsureUserData.
func UserDataDir() string {
	if override := strings.TrimSpace(os.Getenv(EnvUserData)); override != "" {
		return override
	}
	if stored := StoredDataDir(); stored != "" {
		return stored
	}
	return DefaultUserDataDir()
}

// DefaultWorkspace is the first-run project folder when config.workspace is empty.
func DefaultWorkspace() string {
	return filepath.Join(UserDataDir(), "workspace")
}

// DshHome returns the dsh home directory inside the user data directory,
// where dsh keeps its sessions and settings. It does not create it; see
// EnsureDshHome.
func DshHome() string {
	return filepath.Join(UserDataDir(), "dsh-home")
}

// RuntimeDir returns the runtime directory inside the user data
// directory, for files that describe the running process. It does not
// create it; see EnsureRuntimeDir.
func RuntimeDir() string {
	return filepath.Join(UserDataDir(), "runtime")
}

// PidFile returns the file that records the running shell's process id,
// inside the runtime directory.
func PidFile() string {
	return filepath.Join(RuntimeDir(), "whaleisle.pid")
}

// HostRecordFile returns the file that names the Host process this profile's
// shell started, so the next shell can reap one left behind. It sits beside the
// pid file because both describe a process, not user data.
func HostRecordFile() string {
	return filepath.Join(RuntimeDir(), "host-process.json")
}

// EnvVendor, when set, overrides the vendored runtime directory that holds
// deepseek-harness and the desktop plugin packages.
const EnvVendor = "WHALEISLE_VENDOR"

// EnvNodeDir, when set, overrides the bundled Node runtime directory. When both
// WHALEISLE_VENDOR and WHALEISLE_NODE_DIR are unset and a complete pinned runtime
// (see pinnedHostRuntimeDirs) exists, VendorDir prefers it over runtime/vendor.
const EnvNodeDir = "WHALEISLE_NODE_DIR"

func bundledNodeFilename() string {
	if runtime.GOOS == "windows" {
		return "node.exe"
	}
	return "node"
}

// pinnedHostRuntimeDirs lists the complete bundled Host runtimes a source tree
// may carry, newest pin first. A candidate counts only when its vendor tree and
// its own bundled Node are both present: the Host refuses a Node that is not the
// one it was staged with, so pairing is the point, not a convenience.
var pinnedHostRuntimeDirs = [...]string{"runtime-0.2.2-rc5", "runtime-0.2.2-rc2", "runtime-rc4"}

func hostRuntimePairReady(base, dir string) bool {
	harnessDir := filepath.Join(base, dir, "vendor", "deepseek-harness")
	nodePath := filepath.Join(base, dir, "node", bundledNodeFilename())
	hInfo, err := os.Stat(harnessDir)
	if err != nil || !hInfo.IsDir() {
		return false
	}
	nInfo, err := os.Stat(nodePath)
	return err == nil && nInfo.Mode().IsRegular()
}

// BundledHostRuntimeDir names the first complete pinned Host runtime under base,
// in pinnedHostRuntimeDirs order, or "" when none is complete. Vendor selection
// and Node selection both use it so they can never pair mismatched trees.
func BundledHostRuntimeDir(base string) string {
	for _, dir := range pinnedHostRuntimeDirs {
		if hostRuntimePairReady(base, dir) {
			return dir
		}
	}
	return ""
}

// VendorDir returns the directory of vendored harness + desktop plugins
// (preferring a complete pinned runtime when unoverridden, falling back to
// runtime/vendor). Override with WHALEISLE_VENDOR; otherwise use a selected
// macOS Host component, then look next to the executable and working directory.
func VendorDir() string {
	if override := os.Getenv(EnvVendor); override != "" {
		return override
	}
	if selected := SelectedHostRuntime(); selected != "" {
		return filepath.Join(selected, "vendor")
	}
	bases := []string{}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		bases = append(bases, exeDir)
		if strings.EqualFold(filepath.Base(exeDir), "bin") {
			bases = append(bases, filepath.Dir(exeDir))
		}
	}
	if wd, err := os.Getwd(); err == nil {
		bases = append(bases, wd)
	}
	if strings.TrimSpace(os.Getenv(EnvNodeDir)) == "" {
		for _, base := range bases {
			if dir := BundledHostRuntimeDir(base); dir != "" {
				return filepath.Join(base, dir, "vendor")
			}
		}
	}
	for _, base := range bases {
		c := filepath.Join(base, "runtime", "vendor")
		if st, err := os.Stat(filepath.Join(c, "deepseek-harness")); err == nil && st.IsDir() {
			return c
		}
	}
	// Prefer a working-directory candidate over an exe-adjacent miss (test
	// binaries live under GOCACHE/gotmp and have no runtime/vendor tree).
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "runtime", "vendor")
	}
	if len(bases) > 0 {
		return filepath.Join(bases[len(bases)-1], "runtime", "vendor")
	}
	return filepath.Join("runtime", "vendor")
}

// HarnessRoot is the vendored deepseek-harness checkout under VendorDir.
func HarnessRoot() string {
	return filepath.Join(VendorDir(), "deepseek-harness")
}

// HarnessBinJS is the CLI entry used to launch `dsh web` from source.
func HarnessBinJS() string {
	return filepath.Join(HarnessRoot(), "apps", "cli", "lib", "bin.js")
}

// WebProfileDir is the dsh web profile inside DshHome (overlays + junctions).
func WebProfileDir() string {
	return filepath.Join(DshHome(), "profiles", "web")
}

// EnsureUserData creates the user data directory if it is missing.
func EnsureUserData() error {
	return os.MkdirAll(UserDataDir(), 0o755)
}

// EnsureDshHome creates the dsh home directory, and the user data
// directory above it, if they are missing.
func EnsureDshHome() error {
	return os.MkdirAll(DshHome(), 0o755)
}

// EnsureRuntimeDir creates the runtime directory, and the user data
// directory above it, if they are missing.
func EnsureRuntimeDir() error {
	return os.MkdirAll(RuntimeDir(), 0o755)
}
