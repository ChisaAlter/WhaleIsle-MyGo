package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
	t.Setenv(paths.EnvUserData, t.TempDir())
	release := updateCenterFixture()
	s := &mainState{}
	s.nativeReleases = nativeReleaseState{centerLoaded: true, autoDownload: true, paused: true, phase: "paused", active: release, route: "cnb", requireRuntime: true, received: 8, total: 16, staging: nativeReleaseStagingDir(release)}
	s.persistNativeUpdateCenter()
	reopened := &mainState{}
	reopened.ensureNativeUpdateCenter()
	r := reopened.nativeReleases
	if !r.centerLoaded || !r.paused || r.busy || !r.autoDownload || !r.requireRuntime || r.active.URL != release.URL || r.active.DownloadRoute != "github" || r.received != 8 {
		t.Fatalf("recovery=%+v", r)
	}
	if r.staged.Path != "" {
		t.Fatal("partial file became an installer")
	}
	reopened.cancelNativeUpdate(nil)
	reopened2 := &mainState{}
	reopened2.ensureNativeUpdateCenter()
	if reopened2.nativeReleases.active.Tag != "" || reopened2.nativeReleases.paused || !reopened2.nativeReleases.autoDownload {
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
	if s.nativeReleases.err == "" {
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
	if discardNativePartial(outside) == nil {
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
