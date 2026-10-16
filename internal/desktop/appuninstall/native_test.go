package appuninstall

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Only inert temporary files and a fake process starter; no OS action is run.
func TestNativeUninstallFixture(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "whaleisle.exe")
	uninstall := filepath.Join(root, "Uninstall.exe")
	for _, p := range []string{current, uninstall} {
		if err := os.WriteFile(p, []byte("inert fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	record := Record{"Whale Isle", NativeMarker, root, `"` + uninstall + `"`}
	plan, err := Resolve(current, record)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	fake := func(path string) error {
		calls++
		if path != uninstall {
			t.Fatalf("unexpected executable %q", path)
		}
		return nil
	}
	if calls != 0 {
		t.Fatal("discovery launched a process")
	}
	if err := plan.Launch(fake); err != nil || calls != 1 {
		t.Fatalf("fake launch: %v, %d", err, calls)
	}
	for _, change := range []func(*Record){
		func(r *Record) { r.NativeRuntime = "" },
		func(r *Record) { r.NativeRuntime = "Electron" },
		func(r *Record) { r.DisplayName = "Deepseek Harness Desktop" },
		func(r *Record) { r.InstallLocation = filepath.Join(root, "other") },
		func(r *Record) { r.UninstallString += " /S" },
		func(r *Record) { r.UninstallString = "cmd.exe /c " + uninstall },
		func(r *Record) { r.UninstallString = `"` + uninstall + `" & whoami` },
	} {
		bad := record
		change(&bad)
		if _, err := Resolve(current, bad); !errors.Is(err, ErrNotNative) {
			t.Fatalf("accepted mismatched/legacy record: %#v", bad)
		}
	}
	if _, err := Resolve(filepath.Join(root, "source-build.exe"), record); !errors.Is(err, ErrNotNative) {
		t.Fatal("accepted non-product executable")
	}
	if err := plan.Launch(nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil starter: %v", err)
	}
	launchErr := errors.New("fake start failed")
	if err := plan.Launch(func(string) error { return launchErr }); !errors.Is(err, launchErr) {
		t.Fatal("lost process start failure")
	}
	if err := os.Remove(uninstall); err != nil {
		t.Fatal(err)
	}
	if err := plan.Launch(fake); err == nil || calls != 1 {
		t.Fatal("missing/replaced uninstaller dispatched")
	}
	if err := os.Mkdir(uninstall, 0700); err != nil {
		t.Fatal(err)
	}
	if err := plan.Launch(fake); !errors.Is(err, ErrNotNative) || calls != 1 {
		t.Fatal("directory dispatched")
	}
	if err := (Plan{}).Launch(fake); !errors.Is(err, ErrNotNative) {
		t.Fatal("zero plan accepted")
	}
}
