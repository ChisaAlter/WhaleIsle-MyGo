package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"whaleisle/internal/product"
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
	if got, want := DefaultWorkspace(), filepath.Join(dir, "Documents", product.EnglishName); got != want {
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
	if got, want := HostRecordFile(), filepath.Join(dir, "runtime", "host-process.json"); got != want {
		t.Fatalf("HostRecordFile() = %q, want %q", got, want)
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
	if runtime.GOOS != "windows" {
		t.Skip("the AppData override is a Windows contract; elsewhere os.UserConfigDir reads XDG/HOME")
	}
	dir := t.TempDir()
	t.Setenv(EnvUserData, "")
	t.Setenv("AppData", dir) // os.UserConfigDir on Windows reads AppData
	InvalidateDataLocation()
	t.Cleanup(InvalidateDataLocation)
	got := UserDataDir()
	// MyGo's own directory, with the data-location record above it rather
	// than inside it.
	if want := filepath.Join(dir, "WhaleIsle-MyGo", "data"); got != want {
		t.Fatalf("UserDataDir() = %q, want %q", got, want)
	}
}

func TestVendorDirPrefersPairedRC4AndFallsBackToRuntime(t *testing.T) {
	base := t.TempDir()
	t.Chdir(base)
	t.Setenv(EnvVendor, "")
	t.Setenv(EnvNodeDir, "")

	rcHarness := filepath.Join(base, "runtime", "vendor", "deepseek-harness")
	rc4Harness := filepath.Join(base, "runtime-rc4", "vendor", "deepseek-harness")
	rc4Node := filepath.Join(base, "runtime-rc4", "node", bundledNodeFilename())
	if err := os.MkdirAll(rcHarness, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rc4Harness, 0o755); err != nil {
		t.Fatal(err)
	}
	// Missing runtime-rc4 bundled node must fall back to runtime/vendor.
	if got, want := VendorDir(), filepath.Join(base, "runtime", "vendor"); got != want {
		t.Fatalf("incomplete rc4 selected %q, want %q", got, want)
	}
	if err := os.MkdirAll(filepath.Dir(rc4Node), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rc4Node, []byte("node"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := VendorDir(), filepath.Join(base, "runtime-rc4", "vendor"); got != want {
		t.Fatalf("paired rc4 got %q, want %q", got, want)
	}
	// Explicit WHALEISLE_NODE_DIR keeps runtime/vendor ahead of runtime-rc4 auto-switch.
	t.Setenv(EnvNodeDir, filepath.Join(base, "custom-node"))
	if got, want := VendorDir(), filepath.Join(base, "runtime", "vendor"); got != want {
		t.Fatalf("with WHALEISLE_NODE_DIR set, got %q, want %q", got, want)
	}
}

func TestVendorDirPrefersPairedRuntime022OverRC4(t *testing.T) {
	base := t.TempDir()
	t.Chdir(base)
	t.Setenv(EnvVendor, "")
	t.Setenv(EnvNodeDir, "")

	rcHarness := filepath.Join(base, "runtime", "vendor", "deepseek-harness")
	rc4Harness := filepath.Join(base, "runtime-rc4", "vendor", "deepseek-harness")
	rc4Node := filepath.Join(base, "runtime-rc4", "node", bundledNodeFilename())
	rt022Harness := filepath.Join(base, "runtime-0.2.2-rc2", "vendor", "deepseek-harness")
	rt022Node := filepath.Join(base, "runtime-0.2.2-rc2", "node", bundledNodeFilename())
	for _, dir := range []string{rcHarness, rc4Harness, filepath.Dir(rc4Node), rt022Harness} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(rc4Node, []byte("node"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A newer vendor tree without its own bundled Node must not be selected.
	if got, want := VendorDir(), filepath.Join(base, "runtime-rc4", "vendor"); got != want {
		t.Fatalf("incomplete 0.2.2 runtime selected %q, want %q", got, want)
	}
	if err := os.MkdirAll(filepath.Dir(rt022Node), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rt022Node, []byte("node"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := VendorDir(), filepath.Join(base, "runtime-0.2.2-rc2", "vendor"); got != want {
		t.Fatalf("paired 0.2.2 got %q, want %q", got, want)
	}
	// Explicit WHALEISLE_NODE_DIR still keeps runtime/vendor ahead of any switch.
	t.Setenv(EnvNodeDir, filepath.Join(base, "custom-node"))
	if got, want := VendorDir(), filepath.Join(base, "runtime", "vendor"); got != want {
		t.Fatalf("with WHALEISLE_NODE_DIR set, got %q, want %q", got, want)
	}
}
