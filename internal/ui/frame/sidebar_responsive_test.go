package frame

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Installed AppFrame uses a 1024 desktop narrow band independently of the
// separate portrait phone drawer. Neither auto-collapse nor its manual override
// changes the saved width; crossing 1024 resets only that override.
func TestSidebarResponsivePreferenceAndManualOverride(t *testing.T) {
	s := DefaultLayout()
	s.Sidebar = 336
	if c := s.ResolveColumns(1200, false); c.Sidebar != 336 {
		t.Fatal(c)
	}
	for _, width := range []float32{1023, 960, 900, 768} {
		if c := s.ResolveColumns(width, false); c.Sidebar != SidebarCollapsedWidth() || s.Sidebar != 336 || s.drawerMode() {
			t.Fatalf("width %v: desktop narrow=%+v saved=%v drawer=%v", width, c, s.Sidebar, s.drawerMode())
		}
	}
	s.ToggleSidebar()
	if c := s.ResolveColumns(900, false); c.Sidebar != 336 || s.Sidebar != 336 {
		t.Fatal("manual narrow expand lost saved width", c, s.Sidebar)
	}
	if c := s.ResolveColumns(768, false); c.Sidebar != 336 {
		t.Fatal("resize within narrow band reset override", c)
	}
	s.ResolveColumns(1024, false)
	if c := s.ResolveColumns(1023, false); c.Sidebar != SidebarCollapsedWidth() {
		t.Fatal("1024 crossover did not reset override", c)
	}
	if c := s.ResolveColumns(1200, false); c.Sidebar != 336 || s.Sidebar != 336 {
		t.Fatal("wide preference overwritten", c, s.Sidebar)
	}
	// Only an explicit wide-band toggle changes the persistent preference.
	s.ToggleSidebar()
	if c := s.ResolveColumns(1200, false); c.Sidebar != SidebarCollapsedWidth() || s.Sidebar != 0 {
		t.Fatal("wide collapse", c, s.Sidebar)
	}
	s.ResolveColumns(960, false)
	s.ToggleSidebar()
	if c := s.ResolveColumns(960, false); c.Sidebar != SidebarDefault || s.Sidebar != 0 || s.SidebarExpanded != 336 {
		t.Fatal("narrow open changed saved collapsed state", c, s)
	}
	s.ResolveColumns(1200, false)
	s.ToggleSidebar()
	if c := s.ResolveColumns(1200, false); c.Sidebar != SidebarDefault {
		t.Fatal("explicit wide restore did not reset to source default", c, s.SidebarExpanded)
	}
}

func TestSidebarDrawerUsesScreenOrientationBeforeViewportFallback(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		width, height            float32
		landscape, known, drawer bool
	}{
		{"portrait missing screen", 390, 900, false, false, true},
		{"landscape missing screen", 600, 400, false, false, false},
		{"landscape screen tall window", 390, 900, true, true, false},
		{"portrait screen wide window", 600, 400, false, true, true},
		{"768 no drawer", 768, 900, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := DefaultLayout()
			s.observeSidebarOrientation(tc.width, tc.height, tc.landscape, tc.known)
			s.ResolveColumns(tc.width, false) // A width-only resolution preserves the observation.
			if s.drawerMode() != tc.drawer {
				t.Fatalf("drawer=%v want%v %+v", s.drawerMode(), tc.drawer, s.Responsive)
			}
			if !s.sidebarCollapsed() || s.Sidebar != SidebarDefault {
				t.Fatal("narrow initial state changed saved preference")
			}
		})
	}
}

func TestDesktopSidebarResizeAndTransientExpandTester(t *testing.T) {
	for _, width := range []int{768, 960, 1023, 1024} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Sidebar, sizes.Bottom = 336, 0
			store := session.New()
			tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: store, Sizes: &sizes, OnOpenSettings: func() {}}) }, width, 600)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			_, hasDivider := tt.Find("Sidebar divider")
			if hasDivider != (width >= 1024) || sizes.drawerMode() {
				t.Fatalf("width%d divider=%v drawer=%v", width, hasDivider, sizes.drawerMode())
			}
			if width < 1024 {
				sizes.ToggleSidebar()
				tt.Frame()
				divider, ok := tt.Find("Sidebar divider")
				if !ok || divider.X != 336 || sizes.Sidebar != 336 {
					t.Fatalf("manual expand geometry=%+v saved=%v", divider, sizes.Sidebar)
				}
				tt.SetSize(1024, 600)
				tt.SetSize(1023, 600)
				if _, ok := tt.Find("Sidebar divider"); ok || !sizes.sidebarCollapsed() || sizes.Sidebar != 336 {
					t.Fatal("crossover retained transient expansion or changed width")
				}
			}
		})
	}
}

func TestPhoneDrawerAndLandscapeRailTester(t *testing.T) {
	for _, tc := range []struct {
		width, height int
		drawer        bool
	}{{390, 900, true}, {600, 400, false}} {
		sizes := DefaultLayout()
		sizes.Bottom = 0
		tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: session.New(), Sizes: &sizes}) }, tc.width, tc.height)
		tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
		if sizes.drawerMode() != tc.drawer {
			t.Fatalf("%dx%d drawer=%v", tc.width, tc.height, sizes.drawerMode())
		}
		if _, ok := tt.Find(i18n.T(i18n.KeySidebarExpand)); !ok && SidebarCollapsedWidth() != 0 {
			t.Fatal("narrow navigation has no expand control")
		}
		sizes.ToggleSidebar()
		tt.Frame()
		_, track := tt.Find("Sidebar divider")
		if track == tc.drawer || sizes.Sidebar != SidebarDefault {
			t.Fatalf("%dx%d track=%v saved=%v", tc.width, tc.height, track, sizes.Sidebar)
		}
	}
}
