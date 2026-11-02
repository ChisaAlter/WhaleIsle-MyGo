package frame

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestDesktopCaptionDensityInstalledBoundaries(t *testing.T) {
	for _, tc := range []struct {
		center, details float32
		want            string
	}{
		{719, 0, "compact"}, {720, 0, "cozy"}, {899, 0, "cozy"}, {900, 0, "full"},
		{1000, 719, "compact"}, {1000, 720, "cozy"}, {1000, 899, "cozy"}, {1000, 900, "full"}, {719, 950, "compact"},
	} {
		t.Run(fmt.Sprintf("center%v_details%v", tc.center, tc.details), func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Center = tc.center
			got := ""
			ui.NewTester(func(c *ui.Context) {
				m := Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: tc.details > 0, FilePaneWidth: &tc.details}
				m.WorkspaceTrailing = func(c *ui.Context, density string) { got = density; ui.Box(c).Size(68, 32) }
				workspaceSharedCaption(c, m, session.ViewState{}, theme.Resolve("deepseek.light").Light, false)
			}, 1200, 200)
			if got != tc.want {
				t.Fatalf("density=%q want%q", got, tc.want)
			}
		})
	}
}

func TestDesktopNarrowCaptionPanelAndTerminalRemainClickable(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	for _, width := range []int{390, 600, 768, 960, 1023, 1024} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom, sizes.Right = 0, 0
			store := session.New()
			store.Select("caption-session")
			store.AppendStreamDelta("user", "Caption control fixture")
			open, terminalClicks, panelClicks := false, 0, 0
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{Store: store, Sizes: &sizes, RightPaneOpen: open, HideChatViewTabs: true}
				m.WorkspaceTrailing = func(c *ui.Context, density string) {
					chrome.WorkspaceTrailing(c, chrome.WorkspaceTrailingModel{Density: density, HideGit: true, ShowTerminal: true, TerminalAvailable: true, ShowSurfaces: true,
						OnToggleTerminal: func() {
							terminalClicks++
							if sizes.Bottom == 0 {
								sizes.Bottom = 180
							} else {
								sizes.Bottom = 0
							}
						},
						OnToggleSurfaces: func() { panelClicks++; open = !open }})
				}
				View(c, m)
			}, width, 900)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			for _, label := range []string{"Toggle terminal drawer", "Toggle right panel"} {
				r, ok := tt.Find(label)
				if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.X+r.W > float32(width) || r.Y < 0 || r.Y+r.H > 48 {
					t.Fatalf("%q unreachable %+v found%v", label, r, ok)
				}
			}
			title, ok := tt.Find("Conversation session title")
			if !ok || title.Y != 12 || title.H != 32 {
				t.Fatalf("title left caption row %+v found%v", title, ok)
			}
			terminal, _ := tt.Find("Toggle terminal drawer")
			if title.X+title.W > terminal.X {
				t.Fatalf("title overlaps trailing controls: title=%+v terminal=%+v", title, terminal)
			}
			if err := tt.Click("Toggle terminal drawer"); err != nil {
				t.Fatal(err)
			}
			if terminalClicks != 1 || sizes.Bottom != 180 {
				t.Fatal("narrow terminal control did not dispatch")
			}
			if err := tt.Click("Toggle right panel"); err != nil {
				t.Fatal(err)
			}
			if panelClicks != 1 || !open {
				t.Fatal("narrow right panel control did not dispatch")
			}
			if sizes.Sidebar != SidebarDefault {
				t.Fatal("caption gestures changed saved sidebar width")
			}
		})
	}
}
