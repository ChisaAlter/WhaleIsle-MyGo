package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserDataDirOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvUserData, dir)
	if got := UserDataDir(); got != dir {
		t.Fatalf("UserDataDir() = %q, want %q", got, dir)
	}
}

func TestDerivedPaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "user-data")
	t.Setenv(EnvUserData, dir)
	if got, want := DefaultWorkspace(), filepath.Join(dir, "workspace"); got != want {
		t.Fatalf("DefaultWorkspace() = %q, want %q", got, want)
	}
	if got, want := DshHome(), filepath.Join(dir, "dsh-home"); got != want {
		t.Fatalf("DshHome() = %q, want %q", got, want)
	}
	if got, want := RuntimeDir(), filepath.Join(dir, "runtime"); got != want {
		t.Fatalf("RuntimeDir() = %q, want %q", got, want)
	}
	if got, want := PidFile(), filepath.Join(dir, "runtime", "whaleisle.pid"); got != want {
		t.Fatalf("PidFile() = %q, want %q", got, want)
	}
	if got, want := WebProfileDir(), filepath.Join(dir, "dsh-home", "profiles", "web"); got != want {
		t.Fatalf("WebProfileDir() = %q, want %q", got, want)
	}
}

func TestVendorDirOverride(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "deepseek-harness")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvVendor, dir)
	if got := VendorDir(); got != dir {
		t.Fatalf("VendorDir() = %q, want %q", got, dir)
	}
	if got := HarnessRoot(); got != root {
		t.Fatalf("HarnessRoot() = %q, want %q", got, root)
	}
	if got, want := HarnessBinJS(), filepath.Join(root, "apps", "cli", "lib", "bin.js"); got != want {
		t.Fatalf("HarnessBinJS() = %q, want %q", got, want)
	}
}

func TestEnsureCreatesDirs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "user-data")
	t.Setenv(EnvUserData, dir)
	for _, fn := range []func() error{EnsureUserData, EnsureDshHome, EnsureRuntimeDir} {
		if err := fn(); err != nil {
			t.Fatalf("ensure: %v", err)
		}
	}
	for _, name := range []string{"dsh-home", "runtime"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestUserDataDirWithoutOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvUserData, "")
	t.Setenv("AppData", dir) // os.UserConfigDir on Windows reads AppData
	got := UserDataDir()
	if want := filepath.Join(dir, "Deepseek-Harness-Desktop"); got != want {
		t.Fatalf("UserDataDir() = %q, want %q", got, want)
	}
}
