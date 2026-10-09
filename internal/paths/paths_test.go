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
	if got, want := DshHome(), filepath.Join(dir, "dsh-home"); got != want {
		t.Fatalf("DshHome() = %q, want %q", got, want)
	}
	if got, want := RuntimeDir(), filepath.Join(dir, "runtime"); got != want {
		t.Fatalf("RuntimeDir() = %q, want %q", got, want)
	}
	if got, want := PidFile(), filepath.Join(dir, "runtime", "whaleisle.pid"); got != want {
		t.Fatalf("PidFile() = %q, want %q", got, want)
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
