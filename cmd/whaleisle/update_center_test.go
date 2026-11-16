package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"whaleisle/internal/app/updater"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

func updateCenterFixture() update.NativeRelease {
	return update.NativeRelease{Feed: update.Feed{Version: "1.2.3", URL: "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.3/fixture.exe", SHA256: strings.Repeat("a", 64), Size: 16}, Tag: "v1.2.3", Platform: runtime.GOOS, Arch: runtime.GOARCH, Format: "nsis", DownloadRoute: "github"}
}

func TestNativeUpdateCenterRestoresPinnedPauseAndPreference(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("installed native installer recovery baseline is Windows")
	}
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	release := updateCenterFixture()
	s := &mainState{}
	s.updaterCenter().RestoreReleaseState(updater.ReleaseState{
		CenterLoaded: true, AutoDownload: true, Paused: true, Phase: "paused", Active: release,
		Route: "cnb", RequireRuntime: true, Received: 8, Total: 16,
		Staging: filepath.Join(dir, "updates", "native-partials", release.PackageKey()),
	})
	s.updaterCenter().Persist()
	reopened := &mainState{}
	reopened.ensureNativeUpdateCenter()
	r := reopened.updaterCenter().ReleaseSnapshot()
	if !r.CenterLoaded || !r.Paused || r.Busy || !r.AutoDownload || !r.RequireRuntime || r.Active.URL != release.URL || r.Active.DownloadRoute != "github" || r.Received != 8 {
		t.Fatalf("recovery=%+v", r)
	}
	if r.Staged.Path != "" {
		t.Fatal("partial file became an installer")
	}
	reopened.updaterCenter().Cancel(nil)
	reopened2 := &mainState{}
	reopened2.ensureNativeUpdateCenter()
	r2 := reopened2.updaterCenter().ReleaseSnapshot()
	if r2.Active.Tag != "" || r2.Paused || !r2.AutoDownload {
		t.Fatal("cancel or preference did not persist")
	}
}

func TestNativeUpdateCenterPreservesMalformedStateAndBoundsDiscard(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	dir := filepath.Join(paths.UserDataDir(), "updates")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dir, "update-center.json")
	if err := os.WriteFile(name, []byte("invalid fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &mainState{}
	s.ensureNativeUpdateCenter()
	if s.updaterCenter().ReleaseSnapshot().Err == "" {
		t.Fatal("corrupt recovery was hidden")
	}
	preserved, err := filepath.Glob(name + ".invalid-*")
	if err != nil || len(preserved) != 1 {
		t.Fatalf("original not preserved: %v %v", preserved, err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "keep"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if updater.DiscardNativePartial(paths.UserDataDir(), outside) == nil {
		t.Fatal("outside directory was accepted for recursive removal")
	}
	if b, err := os.ReadFile(filepath.Join(outside, "keep")); err != nil || string(b) != "keep" {
		t.Fatal("unrelated recovery material changed")
	}
}

func TestTaskConfirmationRetainsEveryOriginalWorkField(t *testing.T) {
	line := taskWorkDescription(taskcontrol.Work{Kind: "schedule-task", ID: "task-id", Title: "Task title", Owner: "agent:fixture", Detail: "queued", SessionID: "session-id", ScheduledAt: "2026-11-02T00:00:00Z", Due: true, Recurring: true, Count: 2})
	for _, field := range []string{"schedule-task", "task-id", "Task title", "agent:fixture", "queued", "session-id", "2026-11-02T00:00:00Z", "2"} {
		if !strings.Contains(line, field) {
			t.Fatalf("missing %s: %s", field, line)
		}
	}
}
