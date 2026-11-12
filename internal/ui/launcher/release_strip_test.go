package launcher

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// The failed-download state renders its controls inside the same install-card
// slot as the idle CTA, and 清除 dismisses back to that CTA without touching
// any other state.
func TestReleaseStripDismissRestoresIdleSlot(t *testing.T) {
	dismissed := false
	m := func() Model {
		return Model{
			ThemeID: "light", Version: "dev", LauncherPackage: true,
			ActiveTab: TabHome,
			// ReleaseStatus + ReleaseError reproduce the visible strip state
			// observed on a failed desktop install.
			ReleaseStatus:    "Downloading and verifying the installer…",
			ReleaseError:     "no compatible native package for this selection; a full runtime package is required when Host is missing",
			OnDismissRelease: func() { dismissed = true },
			OnInstallRuntime: func() {},
		}
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m()) }, 1025, 703)
	if err := tt.Click(releaseText("清除", "Dismiss")); err != nil {
		t.Fatal(err)
	}
	if !dismissed {
		t.Fatal("dismiss callback not consumed")
	}
}

// A long error message must stay inside the stage column: buttons remain
// findable and nothing overlaps the rail.
func TestReleaseStripLongErrorDoesNotWiden(t *testing.T) {
	longErr := "download failed: read tcp 192.0.2.1:443->198.51.100.7:51322: wsarecv: an existing connection was forcibly closed by the remote host while reading https://objects.example.invalid/release.tar.gz segment 4123 of 9000"
	m := func() Model {
		return Model{
			ThemeID: "light", Version: "dev", LauncherPackage: true,
			ActiveTab:              TabHome,
			ReleaseError:           longErr,
			OnDismissRelease:       func() {},
			OnInstallRuntime:       func() {},
			ReleaseSelectedPackage: "",
		}
	}
	for _, size := range [][2]int{{1025, 703}, {760, 540}, {560, 420}} {
		tt := ui.NewTester(func(c *ui.Context) { View(c, m()) }, size[0], size[1])
		if !tt.HasText(releaseText("清除", "Dismiss")) {
			t.Fatalf("dismiss control not reachable at %dx%d", size[0], size[1])
		}
		if !tt.HasText(releaseText("重试", "Retry")) {
			t.Fatalf("retry control not reachable at %dx%d", size[0], size[1])
		}
	}
}

// Controls backed by callbacks the standalone launcher never wires must not
// render as dead UI: the versions page auto-download checkbox only appears
// when OnSetAutoDownload exists, and the rail route chip only when
// DownloadRoute is bound.
func TestUnwiredControlsDoNotRender(t *testing.T) {
	base := func() Model {
		return Model{ThemeID: "light", Version: "dev", LauncherPackage: true, ActiveTab: TabVersions}
	}
	dead := ui.NewTester(func(c *ui.Context) { View(c, base()) }, 1025, 703)
	if dead.HasText(releaseText("有新版本时在后台自动下载", "Download new versions automatically in the background")) {
		t.Fatal("auto-download checkbox rendered without OnSetAutoDownload")
	}
	if dead.HasText(i18n.T(i18n.KeyLauncherRouteLabel)) {
		t.Fatal("route chip rendered without DownloadRoute")
	}

	m := base()
	m.OnSetAutoDownload = func(bool) {}
	route := "github"
	m.DownloadRoute = &route
	wired := ui.NewTester(func(c *ui.Context) { View(c, m) }, 1025, 703)
	if !wired.HasText(releaseText("有新版本时在后台自动下载", "Download new versions automatically in the background")) {
		t.Fatal("auto-download checkbox missing when OnSetAutoDownload wired")
	}
	if !wired.HasText(i18n.T(i18n.KeyLauncherRouteLabel)) {
		t.Fatal("route chip missing when DownloadRoute bound")
	}
}
