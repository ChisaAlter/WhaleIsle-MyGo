package frame

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSharedCaptionIsOneRowAcrossEveryRoute(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		hero, hideTabs, global bool
		want                   float32
		// hasTitle is false on the routes whose header is the hero row or the
		// global page's blank seat instead of the conversation title row.
		hasTitle bool
	}{
		{"conversation", false, false, false, 48, true},
		{"without tabs", false, true, false, 48, true},
		{"hero", true, false, false, 48, false},
		{"global panel", false, false, true, 48, false},
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
			// titlebar-density.ts: the cluster sits 12px into the 48px band and
			// ends at the native caption reserve.
			if trailing.Y != 12 || trailing.H != 32 || trailing.X+trailing.W != 1200-112 {
				t.Fatalf("native control reserve/trailing: %+v", trailing)
			}
			// The reference's content row is the plate's 12 top / 4 bottom inset
			// around a 32 DIP row (harness-chrome-inject.js:89; ConversationRoot
			// .module.css:49-66,169-173), so the title seat is y = 12..44 and not
			// centred on 24.
			if tc.hasTitle {
				seat, ok := tt.Find("Conversation session title")
				if !ok {
					t.Fatalf("%s: missing conversation title seat", tc.name)
				}
				if seat.Y != 12 || seat.H != 32 {
					t.Fatalf("%s: title seat=%+v, want y=12 h=32 inside the 48 band", tc.name, seat)
				}
				band, ok := tt.Find("Conversation title band")
				if !ok || band.Y != 12 || band.H != 32 {
					t.Fatalf("%s: title band=%+v (found %v), want y=12 h=32", tc.name, band, ok)
				}
			}
			if density != "compact" {
				t.Fatalf("density=%q for center400/details518", density)
			}
		})
	}
}

// Views are a separate, non-draggable row below the native caption, including
// narrow windows where the title actions compact.
func TestConversationViewTabsHaveSeparateRowAndKeyboardNavigation(t *testing.T) {
	for _, width := range []float32{1200, 400} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Center, sizes.Right = width, 0
			tab := "chat"
			store := session.New()
			var band float32
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{Store: store, Sizes: &sizes, ChatPaneTab: &tab}
				ui.Column(c).Fill().Children(func() {
					band = workspaceSharedCaption(c, m, session.ViewState{SelectedID: "s1", Items: []session.Item{{SessionID: "s1", Title: "Current"}}}, theme.Resolve("deepseek.light").Light, false)
					conversationViewTabs(c, m, theme.Resolve("deepseek.light").Light, width < 720)
				})
			}, int(width), 800)
			if band != 48 {
				t.Fatalf("band=%v, want the single 48px caption", band)
			}
			for _, label := range []string{i18n.T(i18n.KeyChatPaneTabChat), i18n.T(i18n.KeyChatPaneTabTrajectory)} {
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing view tab %q; texts=%q", label, tt.Texts())
				}
				if r.Y < 48 || r.Y+r.H > 76 || r.X < 0 || r.X+r.W > width {
					t.Fatalf("view tab %q left the separate 48..76 row: %+v", label, r)
				}
			}
			if err := tt.Click(i18n.T(i18n.KeyChatPaneTabTrajectory)); err != nil {
				t.Fatal(err)
			}
			if tab != "trajectory" {
				t.Fatalf("tab=%q after click", tab)
			}
			tt.Key(0, ui.KeyHome)
			if tab != "chat" || !tt.Focused(i18n.T(i18n.KeyChatPaneTabChat)) {
				t.Fatalf("Home did not choose/focus chat: %q", tab)
			}
			tt.Key(0, ui.KeyEnd)
			if tab != "trajectory" {
				t.Fatalf("End tab=%q", tab)
			}
			saveConversationEvidence(t, tt, fmt.Sprintf("conversation-tabs-%v", width))
		})
	}
}

func TestConversationViewsUseActualHandleAndKeepBuiltins(t *testing.T) {
	tab, opened, cleared := "chat", "", 0
	m := Model{ChatPaneTab: &tab, ConversationViews: []pluginui.ConversationView{
		{Handle: "ignored", ID: "trajectory", Label: "Duplicate"},
		{Handle: "owner:revision:custom", ID: "report", Label: "Report"},
	}, OnSelectConversationView: func(handle string) { opened = handle }, OnClearConversationView: func() { cleared++ }}
	tt := ui.NewTester(func(c *ui.Context) { conversationViewTabs(c, m, theme.Resolve("deepseek.light").Light, false) }, 400, 100)
	if tt.HasText("Duplicate") {
		t.Fatal("duplicated built-in view")
	}
	if err := tt.Click("Report"); err != nil {
		t.Fatal(err)
	}
	if tab != "plugin:owner:revision:custom" || opened != "owner:revision:custom" {
		t.Fatalf("wrong native handle %q %q", tab, opened)
	}
	tt.Key(0, ui.KeyHome)
	if tab != "chat" || cleared != 1 {
		t.Fatalf("builtin did not retire custom: %q %d", tab, cleared)
	}
}

func TestPadReservesBodyWithoutMovingCaptionOrChangingSidebarBreakpoint(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	reserve := float32(320)
	var center, details, trailing ui.Rect
	right := float32(0)
	tt := ui.NewTester(func(c *ui.Context) {
		cols := sizes.resolveColumnsWithReserve(1200, reserve, true)
		right = cols.Details
		m := Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true, FilePaneWidth: &right, RightOverlayReserve: reserve}
		m.WorkspaceTrailing = func(c *ui.Context, _ string) {
			ui.Box(c).Size(120, 32).Draw(func(_ *ui.Painter, r ui.Rect) { trailing = r })
		}
		sidebarColumns(c, &sizes, func() { ui.Box(c).Fill() }, func() {
			centerRightColumns(c, m,
				func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { center = r }) },
				func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { details = r }) },
				func() {
					workspaceSharedCaption(c, m, session.ViewState{}, theme.Resolve("deepseek.light").Light, false)
				})
		})
	}, 1200, 800)
	if sizes.Responsive.narrow {
		t.Fatal("Pad changed full window breakpoint")
	}
	if trailing.X+trailing.W != 1200-112 || center.Y != 48 {
		t.Fatalf("caption shifted: trailing=%+v center=%+v", trailing, center)
	}
	if center.X+center.W > 1200-reserve || (right > 0 && details.X+details.W > 1200-reserve) {
		t.Fatalf("body overlaps Pad center=%+v details=%+v", center, details)
	}
	if center.W != sizes.Center {
		t.Fatalf("resolved center=%v actual=%v", sizes.Center, center.W)
	}
	saveConversationEvidence(t, tt, "pad-body-reserve")
}

func TestCaptionReflowsAfterAsyncToolArrivalAndRemoval(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 600
	width := float32(0)
	store := session.New()
	snap := session.ViewState{
		SelectedID: "s1",
		Items: []session.Item{
			{SessionID: "s1", Title: "Long conversation session title that concedes to trailing tools"},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Store: store, Sizes: &sizes, HideChatViewTabs: true}
		m.WorkspaceTrailing = func(c *ui.Context, _ string) { ui.Box(c).Size(width, 32) }
		workspaceSharedCaption(c, m, snap, theme.Resolve("deepseek.light").Light, false)
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
	if content.Y != 48 || content.H != 400 {
		t.Fatalf("global content=%+v, want below48 caption and above retained400 split", content)
	}
	if _, ok := tt.Find(i18n.T(i18n.KeyTerminalUnavailable)); !ok {
		t.Fatal("global page lost its terminal drawer")
	}
	tt.SetSize(1200, 400)
	// 352 DIP of centre column keeps the terminal's 180 DIP floor, so the chat
	// above the divider gives up the rest (172) without rewriting the split.
	if content.Y != 48 || content.H != 172 || sizes.Main != 400 {
		t.Fatalf("global resize did not concede without rewriting split: content=%+v Main=%v", content, sizes.Main)
	}
	status, ok := tt.Find(i18n.T(i18n.KeyTerminalUnavailable))
	if !ok || status.Y+status.H > 400 {
		t.Fatalf("global terminal escaped resized viewport: %+v", status)
	}
	tt.SetSize(1200, 800)
	if content.Y != 48 || content.H != 400 || sizes.Main != 400 {
		t.Fatalf("global window growth retained a stale concession: content=%+v Main=%v", content, sizes.Main)
	}
	sizes.Bottom = 0
	tt.Frame()
	if content.Y != 48 || content.H != 752 {
		t.Fatalf("closed drawer did not return allocated space: %+v", content)
	}
}
