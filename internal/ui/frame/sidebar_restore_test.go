package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/sidebar"
)

func TestDesktopSidebarRestoreCaption(t *testing.T) {
	for _, platform := range []string{"windows", "darwin"} {
		t.Run(platform, func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Sidebar, sizes.SidebarExpanded = 0, 336
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					drawSidebarRestoreCaption(c, &sizes, chrome.DefaultTitleBarColors(), nil, platform)
					ui.Text(c, "Global settings") // Also usable when NavigationShell is bypassed.
				})
			}, 1200, 800)
			tt.SetTitleBar(ui.TitleBar{Height: 32, Left: 72, Right: 138})
			tt.Frame()
			button, ok := tt.Find(i18n.T(i18n.KeySidebarExpand))
			if platform == "windows" {
				if ok || sizes.Sidebar != 0 {
					t.Fatal("Whale's generic Windows rail must not inject standalone caption controls")
				}
				return
			}
			if !ok {
				t.Fatal("missing caption restore outside sidebar")
			}
			if button.X < 72 || button.X+button.W > 1200-138 || button.Y < 0 || button.Y+button.H > 32 {
				t.Fatalf("unsafe caption button: %+v", button)
			}
			if err := tt.Click(i18n.T(i18n.KeySidebarExpand)); err != nil {
				t.Fatal(err)
			}
			if sizes.Sidebar != SidebarDefault {
				t.Fatalf("wide restore did not reset to source default: %+v", sizes)
			}
		})
	}
}

func TestSidebarCollapsedPlatformContract(t *testing.T) {
	for platform, want := range map[string]float32{"windows": 56, "darwin": 0, "linux": 56} {
		if got := sidebarCollapsedWidth(platform); got != want {
			t.Fatalf("%s collapsed=%v want %v", platform, got, want)
		}
	}
}

func TestCollapsedCaptionCanCreateSessionWithoutExpanding(t *testing.T) {
	for _, platform := range []string{"windows", "darwin"} {
		t.Run(platform, func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Sidebar, sizes.SidebarExpanded = 0, 336
			created := 0
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					drawSidebarRestoreCaption(c, &sizes, chrome.DefaultTitleBarColors(), nil, platform, func() { created++ })
				})
			}, 1200, 800)
			tt.SetTitleBar(ui.TitleBar{Height: 32, Left: 72, Right: 138})
			if platform == "windows" {
				if tt.HasText(i18n.T(i18n.KeySidebarNewSessionFallback)) || created != 0 {
					t.Fatal("Windows rail session action must not be duplicated in the caption")
				}
				return
			}
			if err := tt.Click(i18n.T(i18n.KeySidebarNewSessionFallback)); err != nil {
				t.Fatal(err)
			}
			if created != 1 || sizes.sidebarPreference() != 0 {
				t.Fatal("new-session action must remain available without expanding navigation")
			}
		})
	}
}

func TestWhaleWindowsRailRetainsOriginalActions(t *testing.T) {
	query, focus, collapsed := "", false, true
	created, market, schedule, assistant, browsed := 0, 0, 0, 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		width := float32(56)
		if !collapsed {
			width = 280
		}
		ui.Column(c).Width(width).FillHeight().Children(func() {
			sidebar.View(c, sidebar.Model{
				Collapsed: collapsed, SearchQuery: &query, SearchFocus: &focus,
				OnToggle: func() { collapsed = !collapsed },
				OnCreate: func() { created++ }, OnOpenMarket: func() { market++ },
				OnOpenSchedule: func() { schedule++ }, OnOpenAssistant: func() { assistant++ },
				OnBrowseWorkspace: func() { browsed++ }, OnOpenSettings: func() {},
			})
		})
	}, 360, 800)
	for _, key := range []string{i18n.KeySidebarExpand, i18n.KeySidebarNewSessionFallback, i18n.KeySidebarMarket, i18n.KeySidebarSchedule, i18n.KeySidebarAssistant, i18n.KeySidebarBrowseWorkspace, i18n.KeySidebarSearch} {
		r, ok := tt.Find(i18n.T(key))
		if !ok || r.X < 0 || r.X+r.W > 56 || r.W != 36 || r.H != 36 {
			t.Fatalf("rail control %q outside original36/56 geometry: %+v", key, r)
		}
	}
	for _, key := range []string{i18n.KeySidebarNewSessionFallback, i18n.KeySidebarMarket, i18n.KeySidebarSchedule, i18n.KeySidebarAssistant} {
		if err := tt.Click(i18n.T(key)); err != nil {
			t.Fatal(err)
		}
	}
	if !collapsed || created != 1 || market != 1 || schedule != 1 || assistant != 1 || focus {
		t.Fatal("rail actions expanded navigation, lost callbacks, or stole search focus")
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarBrowseWorkspace)); err != nil {
		t.Fatal(err)
	}
	if collapsed || browsed != 1 || focus {
		t.Fatal("Add did not expand and dispatch its original workspace action")
	}
	collapsed = true
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	if collapsed || !tt.Focused(i18n.T(i18n.KeySidebarSearch)) {
		t.Fatal("explicit rail Search did not focus the expanded shared input")
	}
}

// TestCaptionRestorePreservesNarrowPreference covers the drawer band: the
// caption's expand button opens the drawer transiently and never rewrites the
// saved 336-DIP preference.
func TestCaptionRestorePreservesNarrowPreference(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Sidebar = 336
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			drawSidebarRestoreCaption(c, &sizes, chrome.DefaultTitleBarColors(), nil, "darwin")
		})
	}, 700, 800)
	if err := tt.Click(i18n.T(i18n.KeySidebarExpand)); err != nil {
		t.Fatal(err)
	}
	if sizes.Sidebar != 336 || sizes.sidebarPreference() != 336 {
		t.Fatalf("drawer restore changed preference: %+v", sizes)
	}
	sizes.ToggleSidebar()
	if sizes.Sidebar != 336 || sizes.sidebarPreference() != 0 {
		t.Fatal("drawer override did not close independently")
	}
	sizes.observeSidebarViewport(1200)
	if sizes.sidebarPreference() != 336 {
		t.Fatal("wide preference not restored")
	}
}
