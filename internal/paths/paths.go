// Package paths resolves the file system locations Whale Isle uses:
// the user data directory it shares with the Electron shell, the dsh home
// inside it, and the runtime directory holding the pid file.
//
// Every function here is pure path computation and may be called from any
// goroutine; nothing touches the file system until one of the Ensure
// helpers is called. Tests override the user data directory with
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
