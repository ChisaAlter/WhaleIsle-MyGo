// Package paths resolves the file system locations Whale Isle uses:
// the user data directory it shares with the Electron shell, the dsh home
// inside it, and the runtime directory holding the pid file.
//
// Path lookup may read a managed runtime selection or probe candidate files;
// it never starts a runtime or changes the selection. Tests override user data with
// WHALEISLE_USER_DATA (see UserDataDir).
package paths

import (
	"os"
	"path/filepath"

	"whaleisle/internal/config"
)

// EnvUserData, when set, overrides the user data directory. Tests use it
// to point the paths at a temporary directory.
const EnvUserData = "WHALEISLE_USER_DATA"

// UserDataDir returns the user data directory: the WHALEISLE_USER_DATA
// override when set, else the Electron shell's directory under the user's
// configuration directory (%AppData%\Deepseek-Harness-Desktop on Windows).
// It does not create the directory; see EnsureUserData.
func UserDataDir() string {
	if override := os.Getenv(EnvUserData); override != "" {
		return override
	}
	base, err := os.UserConfigDir()
	if err != nil {
		// os.UserConfigDir fails only when neither the platform's
		// well-known variable nor the home directory resolves; fall
		// back to a directory under the executable's own name so the
		// app still has somewhere to write.
		wd, _ := os.Getwd()
		return filepath.Join(wd, ".whaleisle")
	}
	return filepath.Join(base, config.DataDirName)
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

// EnvVendor, when set, overrides the vendored runtime directory that holds
// deepseek-harness and the desktop plugin packages.
const EnvVendor = "WHALEISLE_VENDOR"

// VendorDir returns the directory of vendored harness + desktop plugins
// (runtime/vendor). Override with WHALEISLE_VENDOR; otherwise use a selected
// macOS Host component, then look next to the executable and working directory.
func VendorDir() string {
	if override := os.Getenv(EnvVendor); override != "" {
		return override
	}
	if selected := SelectedHostRuntime(); selected != "" {
		return filepath.Join(selected, "vendor")
	}
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "runtime", "vendor"))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "runtime", "vendor"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(filepath.Join(c, "deepseek-harness")); err == nil && st.IsDir() {
			return c
		}
	}
	// Prefer a working-directory candidate over an exe-adjacent miss (test
	// binaries live under GOCACHE/gotmp and have no runtime/vendor tree).
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "runtime", "vendor")
	}
	if len(candidates) > 0 {
		return candidates[len(candidates)-1]
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
