package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo"

	"whaleisle/internal/config"
	"whaleisle/internal/paths"
)

func TestIsolatedDesktopDoesNotApplySystemIntegration(t *testing.T) {
	t.Setenv("WHALEISLE_ISOLATED", "1")
	calls := 0
	apply := func() error { calls++; return errors.New("sentinel") }
	if err := desktopSystemIntegration(apply); err != nil || calls != 0 {
		t.Fatalf("isolated run invoked system registration: calls=%d err=%v", calls, err)
	}
	t.Setenv("WHALEISLE_ISOLATED", "")
	if err := desktopSystemIntegration(apply); err == nil || calls != 1 {
		t.Fatalf("normal run lost system registration result: calls=%d err=%v", calls, err)
	}
}

func TestIsolatedDesktopRequiresExplicitNonInstalledData(t *testing.T) {
	t.Setenv("WHALEISLE_ISOLATED", "1")
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{"", "relative-data", filepath.Join(base, config.DataDirName)} {
		t.Setenv(paths.EnvUserData, data)
		if err := prepareIsolatedDesktop(); err == nil {
			t.Fatalf("accepted non-isolated data directory: %q", data)
		}
	}
}

func TestIsolatedDesktopSeparatesNativePathsLockNameAndComponentDiscovery(t *testing.T) {
	t.Setenv("WHALEISLE_ISOLATED", "1")
	data := t.TempDir()
	t.Setenv(paths.EnvUserData, data)
	originalApp := mygo.App
	mygo.App = &mygo.Application{}
	t.Cleanup(func() { mygo.App = originalApp })
	mygo.App.SetName("MyGo鲸屿")
	if err := prepareIsolatedDesktop(); err != nil {
		t.Fatal(err)
	}
	firstName := mygo.App.Name()
	if firstName == "MyGo鲸屿" || !strings.HasPrefix(firstName, "MyGo鲸屿 Isolated ") {
		t.Fatalf("shared installed single-instance identity: %q", firstName)
	}
	for key, want := range map[mygo.PathName]string{
		mygo.PathUserData: filepath.Join(data, "native"),
		mygo.PathCache:    filepath.Join(data, "native", "cache"),
		mygo.PathLogs:     filepath.Join(data, "native", "logs"),
	} {
		got, err := mygo.App.Path(key)
		if err != nil || got != want {
			t.Fatalf("native %s escaped isolated directory: %q %v", key, got, err)
		}
	}
	if roots := desktopComponentRoots(); !reflect.DeepEqual(roots, []string{data}) {
		t.Fatalf("component discovery included installed data: %#v", roots)
	}
	t.Setenv(paths.EnvUserData, t.TempDir())
	if err := prepareIsolatedDesktop(); err != nil || mygo.App.Name() == firstName {
		t.Fatalf("separate isolated directories share a lock identity: %q %v", mygo.App.Name(), err)
	}
}

func TestIsolatedDesktopDoesNotConsumeSharedAcceptanceHook(t *testing.T) {
	t.Setenv("WHALEISLE_ISOLATED", "1")
	t.Setenv("TEMP", t.TempDir())
	t.Setenv("TMP", os.Getenv("TEMP"))
	hook := acceptOpenHookPath()
	if err := os.WriteFile(hook, []byte("settings"), 0o600); err != nil {
		t.Fatal(err)
	}
	var s mainState
	if s.pollAcceptOpenHook() {
		t.Fatal("isolated run accepted a shared hook")
	}
	if b, err := os.ReadFile(hook); err != nil || string(b) != "settings" {
		t.Fatalf("isolated run consumed a shared hook: %q %v", b, err)
	}
}

func TestPortableDesktopKeepsIsolationWithoutHidingWindow(t *testing.T) {
	portableTestEnvironment(t)
	portableDesktopRoot = t.TempDir()
	t.Setenv(paths.EnvUserData, filepath.Join(portableDesktopRoot, "data"))
	if !isolatedDesktopRun() || isolatedDesktopHidden() {
		t.Fatal("portable double-click must be isolated and visible")
	}
	called := false
	if err := desktopSystemIntegration(func() error { called = true; return nil }); err != nil || called {
		t.Fatal("portable run reached system registration")
	}
	if roots := desktopComponentRoots(); !reflect.DeepEqual(roots, []string{paths.UserDataDir()}) {
		t.Fatalf("portable component roots include installed data: %#v", roots)
	}
	t.Setenv("WHALEISLE_ISOLATED", "1")
	if !isolatedDesktopHidden() {
		t.Fatal("explicit QA isolation no longer hides the portable window")
	}
}
