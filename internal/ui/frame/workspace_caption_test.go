package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSharedCaptionKeepsResourceTabsBelowTheWholeHeader(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		hero, hideTabs, global bool
		want                   float32
	}{
		{"conversation", false, false, false, 72},
		{"without tabs", false, true, false, 53},
		{"hero", true, false, false, 52},
		{"global panel", false, false, true, 52},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom, sizes.Right = 0, 540
			store := session.New()
			right := float32(0)
			var center, details, trailing ui.Rect
			var density string
			tt := ui.NewTester(func(c *ui.Context) {
				cols := sizes.ResolveColumns(1200, true)
				right = cols.Details
				m := Model{Store: store, Sizes: &sizes, RightPaneOpen: true, FilePaneWidth: &right, HideChatViewTabs: tc.hideTabs, WorkspacePage: tc.global}
				m.WorkspaceTrailing = func(c *ui.Context, d string) {
					density = d
					ui.Box(c).Size(120, 32).Draw(func(_ *ui.Painter, r ui.Rect) { trailing = r })
				}
				sidebarColumns(c, &sizes, func() { ui.Box(c).Fill() }, func() {
					centerRightColumns(c, m,
						func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { center = r }) },
						func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { details = r }) },
						func() {
							workspaceSharedCaption(c, m, session.ViewState{}, theme.Resolve("deepseek.light").Light, tc.hero)
						})
				})
			}, 1200, 800)
			_ = tt
			if center.Y != tc.want || details.Y != tc.want || center.H != 800-tc.want || details.H != 800-tc.want {
				t.Fatalf("header %s: center=%+v details=%+v, want top%v/full remaining height", tc.name, center, details, tc.want)
			}
			if trailing.Y != 8 || trailing.H != 32 || trailing.X+trailing.W != 1200-112 {
				t.Fatalf("native control reserve/trailing: %+v", trailing)
			}
			if density != "compact" {
				t.Fatalf("density=%q for center400/details518", density)
			}
		})
	}
}

func TestCaptionReflowsAfterAsyncToolArrivalAndRemoval(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 600
	width := float32(0)
	store := session.New()
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Store: store, Sizes: &sizes, HideChatViewTabs: true}
		m.WorkspaceTrailing = func(c *ui.Context, _ string) { ui.Box(c).Size(width, 32) }
		workspaceSharedCaption(c, m, session.ViewState{}, theme.Resolve("deepseek.light").Light, false)
	}, 600, 300)
	before, ok := tt.Find("Conversation session title")
	if !ok {
		t.Fatal("missing title seat")
	}
	width = 220 // Status arrives after the first mounted frame.
	tt.Frame()
	after, _ := tt.Find("Conversation session title")
	if after.W >= before.W || after.X+after.W > 600-112-width-24 {
		t.Fatalf("title did not concede to new tool geometry: before=%+v after=%+v", before, after)
	}
	width = 0 // Both panel preferences and Git can leave an empty trailing row.
	tt.Frame()
	restored, _ := tt.Find("Conversation session title")
	if restored != before {
		t.Fatalf("empty trailing retained stale reserve: before=%+v restored=%+v", before, restored)
	}
}

func TestGlobalPanelKeepsTerminalDrawerAndItsGeometry(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Main = 400
	var content ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		NavigationShell(c, Model{Store: session.New(), Sizes: &sizes, WorkspacePage: true}, func() {
			ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { content = r })
		})
	}, 1200, 800)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	tt.Frame()
	if content.Y != 52 || content.H != 400 {
		t.Fatalf("global content=%+v, want below52 caption and above retained400 split", content)
	}
	if _, ok := tt.Find(i18n.T(i18n.KeyTerminalUnavailable)); !ok {
		t.Fatal("global page lost its terminal drawer")
	}
	sizes.Bottom = 0
	tt.Frame()
	if content.Y != 52 || content.H != 748 {
		t.Fatalf("closed drawer did not return allocated space: %+v", content)
	}
}
