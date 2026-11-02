package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestPortraitDrawerInstalledMinimumAndBackdrop(t *testing.T) {
	for _, tc := range []struct{ viewport, want float32 }{{240, 264}, {264, 264}, {300, 264}, {311, 264}, {312, 264}, {313, 265}, {390, 320}} {
		sizes := DefaultLayout()
		sizes.observeSidebarOrientation(tc.viewport, 900, false, true)
		if got := sizes.drawerWidth(); got != tc.want || !sizes.drawerMode() {
			t.Fatalf("portrait width%v drawer=%v want%v mode=%v", tc.viewport, got, tc.want, sizes.drawerMode())
		}
	}
	// At 300 DIP the 264 floor leaves a 36 DIP backdrop. The actual drawer
	// paints at that width and a pointer click outside it still closes it.
	sizes := DefaultLayout()
	sizes.observeSidebarOrientation(300, 900, false, true)
	sizes.ToggleSidebar()
	var painted ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		sizes.observeSidebarOrientation(300, 900, false, true)
		ui.Column(c).Fill().Children(func() {
			sidebarDrawer(c, &sizes, func() {
				ui.Box(c).Fill().Label("Minimum drawer navigation").Draw(func(_ *ui.Painter, r ui.Rect) { painted = r })
			}, func() { ui.Box(c).Fill() }, 300)
		})
	}, 300, 900)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	if painted.W != 264 || painted.X != 0 {
		t.Fatalf("drawer painted=%+v", painted)
	}
	backdrop, ok := tt.Find(i18n.T(i18n.KeySidebarCollapse))
	if !ok || backdrop.W != 300 {
		t.Fatalf("backdrop=%+v found=%v", backdrop, ok)
	}
	tt.ClickAt(285, 450)
	if sizes.Responsive.expanded || sizes.Sidebar != SidebarDefault {
		t.Fatal("minimum drawer backdrop failed or changed saved sidebar")
	}
	if _, ok := tt.Find("Minimum drawer navigation"); ok {
		t.Fatal("closed minimum drawer retained navigation")
	}
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarExpand)); !ok {
		t.Fatal("closed minimum drawer lost opener")
	}
}
