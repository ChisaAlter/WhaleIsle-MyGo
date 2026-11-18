package main

import (
	"testing"

	"whaleisle/internal/app/updater"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

func TestDisableAutoUpdateEnvSuppressesBackgroundAndAutoDownload(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	t.Setenv("DSH_HOME", t.TempDir())
	t.Setenv(updater.EnvDisableAutoUpdate, "1")

	s := &mainState{}
	s.startNativeUpdateCenter(nil)

	if s.updaterCenter().BackgroundActive() {
		t.Fatal("startNativeUpdateCenter started background loop when WHALEISLE_DISABLE_AUTO_UPDATE=1")
	}

	s.updaterCenter().RestoreReleaseState(updater.ReleaseState{
		CenterLoaded: true,
		Initialized:  true,
		AutoDownload: true,
		Route:        "github",
		CatalogRoute: "github",
		Rows: []update.NativeRelease{
			{Tag: "v99.0.0", Version: "99.0.0", URL: "https://example.invalid/installer.exe", RuntimeIncluded: true},
		},
	})
	s.updaterCenter().MaybeAutoDownload(nil)
	snap := s.updaterCenter().FrameSnapshot()
	if snap.Busy || snap.Phase != "" {
		t.Fatalf("MaybeAutoDownload initiated auto-download while disabled: busy=%v phase=%q", snap.Busy, snap.Phase)
	}
}
