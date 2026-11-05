package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"whaleisle/internal/paths"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
)

func TestAboutUpdateFirstRenderRestoresPausedOwnerWithoutHost(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("persisted native installer fixture uses the installed Windows baseline")
	}
	t.Setenv(paths.EnvUserData, t.TempDir())
	seed := &mainState{}
	release := updateCenterFixture()
	seed.nativeReleases = nativeReleaseState{centerLoaded: true, autoDownload: true, requireRuntime: true, paused: true, phase: "paused", active: release, received: 8, total: 16}
	seed.persistNativeUpdateCenter()
	reopened := &mainState{}
	m := settings.Model{Store: settingsstore.New(), AboutActive: true, About: &settings.AboutAccess{}}
	reopened.wireSettingsBodies(&m)
	if !reopened.nativeReleases.centerLoaded || !reopened.nativeReleases.autoDownload || !reopened.nativeReleases.requireRuntime || m.About.UpdatePhase != "paused" || m.About.UpdatePercent != 50 {
		t.Fatalf("About took an unrestored update snapshot: owner=%+v about=%+v", reopened.nativeReleases, m.About)
	}
	if m.About.OnOpenUpdateCenter == nil || m.About.OnResumeUpdate == nil || m.About.OnDownloadUpdate == nil || m.About.OnInstallUpdate == nil {
		t.Fatal("Host absence hid native update/recovery actions")
	}
}

func TestAboutUpdatePreservesRecoveryAndCatalogErrors(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	t.Setenv(paths.EnvUserData, t.TempDir())
	dir := filepath.Join(paths.UserDataDir(), "updates")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "update-center.json"), []byte("synthetic malformed update state"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &mainState{}
	m := settings.Model{Store: settingsstore.New(), AboutActive: true, About: &settings.AboutAccess{}}
	s.wireSettingsBodies(&m)
	if m.About.UpdatePhase != "error" || !strings.Contains(m.About.UpdateStatus, "Could not read update recovery state") || m.About.OnResumeUpdate == nil || m.About.OnOpenUpdateCenter == nil {
		t.Fatalf("recovery error or recovery entry was hidden: %+v", m.About)
	}
	s.nativeReleases.status = "Catalog lookup failed; existing list retained"
	s.nativeReleases.err = "synthetic catalog failure"
	s.wireSettingsBodies(&m)
	if !strings.Contains(m.About.UpdateStatus, "Catalog lookup failed") || !strings.Contains(m.About.UpdateStatus, "synthetic catalog failure") {
		t.Fatalf("catalog refusal was filtered out: %+v", m.About)
	}
}
