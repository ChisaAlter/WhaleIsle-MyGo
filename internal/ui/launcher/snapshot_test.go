package launcher

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/ui/i18n"
)

// Optional software-rendered evidence; this is not live Windows/Electron parity.
func TestLauncherSoftwareSnapshots(t *testing.T) {
	dir := os.Getenv("LAUNCHER_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set LAUNCHER_SNAPSHOT_DIR to export software renders")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	rows := []components.Row{
		{ID: "one", Name: "Whale Bridge", Description: "Connect your model providers", InstalledVersion: "1.2.3", State: "stopped"},
		{ID: "two", Name: "Sample Plugin", Description: "A second installed component", InstalledVersion: "2.0.0", State: "running"},
		{ID: "three", Name: "Another Plugin", Description: "A third installed component", InstalledVersion: "1.0.0", State: "stopped"},
	}
	save := func(name string, tt *ui.Tester) {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for _, width := range []int{700, 1060, 1400} {
		for _, dark := range []bool{false, true} {
			route, open := "github", false
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{ActiveTab: TabHome, SystemDark: dark, VendorReady: true, Version: "0.3.3", DownloadRoute: &route, RoutePopOpen: &open, Components: rows})
			}, width, 660)
			save(fmt.Sprintf("home-%d-dark-%v.png", width, dark), tt)
		}
	}
	route, open := "github", false
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{ActiveTab: TabHome, VendorReady: true, Version: "0.3.3", DownloadRoute: &route, RoutePopOpen: &open, Components: rows})
	}, 1060, 660)
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteLabel)); err != nil {
		t.Fatal(err)
	}
	save("footer-route-open.png", tt)
	tt = ui.NewTester(func(c *ui.Context) {
		View(c, Model{ActiveTab: TabVersions, VendorReady: true, Version: "0.3.3", DownloadRoute: &route})
	}, 1060, 660)
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteCurrentPrefix) + " GitHub"); err != nil {
		t.Fatal(err)
	}
	save("header-route-open.png", tt)
	tt = ui.NewTester(func(c *ui.Context) {
		View(c, Model{ActiveTab: TabVersions, VendorReady: true, SystemDark: true, Version: "0.3.3", DownloadRoute: &route})
	}, 1060, 660)
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteCurrentPrefix) + " GitHub"); err != nil {
		t.Fatal(err)
	}
	save("header-route-open-dark.png", tt)
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteCNB)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyLauncherRouteCurrentPrefix) + " CNB"); err != nil {
		t.Fatal(err)
	}
	save("header-route-cnb-open-dark.png", tt)
	for i := 3; i < 20; i++ {
		rows = append(rows, components.Row{ID: fmt.Sprint(i), Name: fmt.Sprintf("Component-%02d", i), State: "available"})
	}
	tt = ui.NewTester(func(c *ui.Context) { View(c, Model{ActiveTab: TabComponents, VendorReady: true, Components: rows}) }, 1060, 660)
	tt.Scroll(800, 450, 0, 100000)
	save("components-scrolled-bottom.png", tt)
}
