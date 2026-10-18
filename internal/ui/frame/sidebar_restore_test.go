package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
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
			if !ok {
				t.Fatal("missing caption restore outside sidebar")
			}
			if button.X < 72 || button.X+button.W > 1200-138 || button.Y < 0 || button.Y+button.H > 32 {
				t.Fatalf("unsafe caption button: %+v", button)
			}
			if err := tt.Click(i18n.T(i18n.KeySidebarExpand)); err != nil {
				t.Fatal(err)
			}
			if sizes.Sidebar != 336 {
				t.Fatalf("restore lost width: %+v", sizes)
			}
		})
	}
}

func TestSidebarCollapsedPlatformContract(t *testing.T) {
	for platform, want := range map[string]float32{"windows": 0, "darwin": 0, "linux": 56} {
		if got := sidebarCollapsedWidth(platform); got != want {
			t.Fatalf("%s collapsed=%v want %v", platform, got, want)
		}
	}
}

func TestCaptionRestorePreservesNarrowPreference(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Sidebar = 336
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			drawSidebarRestoreCaption(c, &sizes, chrome.DefaultTitleBarColors(), nil, "darwin")
		})
	}, 900, 800)
	if err := tt.Click(i18n.T(i18n.KeySidebarExpand)); err != nil {
		t.Fatal(err)
	}
	if sizes.Sidebar != 336 || sizes.sidebarPreference() != 336 {
		t.Fatalf("narrow restore changed preference: %+v", sizes)
	}
	sizes.ToggleSidebar()
	if sizes.Sidebar != 336 || sizes.sidebarPreference() != 0 {
		t.Fatal("narrow override did not close independently")
	}
	sizes.observeSidebarViewport(1200)
	if sizes.sidebarPreference() != 336 {
		t.Fatal("wide preference not restored")
	}
}
