package frame

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	settingsui "whaleisle/internal/ui/settings"
)

// Render through both entry points: direct frame and the NavigationShell used
// by main with ExternalNavigation. The body bounds come from the real painter.
type autoFullscreenFrame struct {
	tt           *ui.Tester
	store        *session.Store
	sizes        LayoutSizes
	w            *WorkspacePanes
	open         bool
	route        string
	body         ui.Rect
	closes       int
	settingsOpen bool
	settingsBody ui.Rect
	draft        string
	client       *composer.DraftClient
	chatList     ui.ListState
}

func newAutoFullscreenFrame(t *testing.T, external bool, width, height int) *autoFullscreenFrame {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	f := &autoFullscreenFrame{store: session.New(), sizes: DefaultLayout(), route: "custom-panels"}
	f.w = &WorkspacePanes{}
	f.w.syncRoute(f.route)
	f.sizes.Workspace = f.w
	f.sizes.Bottom = 0
	f.sizes.Right = 540
	f.client = composer.NewDraftClient(&f.draft)
	f.tt = ui.NewTester(func(c *ui.Context) {
		f.body = ui.Rect{}
		m := Model{Store: f.store, Sizes: &f.sizes, RightPaneOpen: f.open, RightPaneTab: &f.route, ExternalNavigation: external,
			Draft: &f.draft, UseComposerTextInput: true, ComposerTextInput: f.client, ChatList: &f.chatList,
			OnToggleTools: func() { f.open = !f.open; f.closes++ },
			WorkspaceTrailing: func(c *ui.Context, _ string) {
				if ui.Button(c, "Toggle rightbar").Clicked() {
					f.open = !f.open
				}
			},
			CustomPanelContent: func(c *ui.Context) {
				ui.Box(c).Fill().MinHeight(0).Label("Ordinary rightbar body").Draw(func(_ *ui.Painter, r ui.Rect) { f.body = r })
			},
		}
		if external {
			NavigationShell(c, m, func() { View(c, m) })
		} else {
			View(c, m)
		}
		// main paints settings as a modal after the retained workspace shell;
		// settingsOpen does not turn navigationModel.WorkspacePage on.
		f.settingsBody = ui.Rect{}
		if f.settingsOpen {
			settingsui.ModalView(c, settingsui.Model{ActiveSection: settingsui.SectionGeneral, OnClose: func() { f.settingsOpen = false }, SectionContent: func(c *ui.Context) {
				ui.Box(c).FillWidth().Height(80).Label("Actual settings modal body").Draw(func(_ *ui.Painter, r ui.Rect) { f.settingsBody = r })
			}})
		}
	}, width, height)
	f.tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	t.Cleanup(func() {
		f.w.Root.eachResource(func(n *WorkspacePane) {
			if n.Files != nil {
				n.Files.Tabs.Dispose()
			}
		})
	})
	return f
}

func requireAutoFrame(t *testing.T, f *autoFullscreenFrame, width int) {
	t.Helper()
	header, ok := f.tt.Find("Workspace pane header")
	if !f.open || !ok || header.X != 0 || header.W != float32(width) || f.body.X != 0 || f.body.W != float32(width) || f.body.H <= 0 {
		t.Fatalf("phone dock did not cover viewport: open=%v header=%+v body=%+v texts=%q", f.open, header, f.body, f.tt.Texts())
	}
	if _, ok := f.tt.Find("Restore pane"); !ok {
		t.Fatal("auto fullscreen did not expose exit control")
	}
	if _, ok := f.tt.Find("Maximize pane"); ok {
		t.Fatal("auto fullscreen exposed enter control")
	}
	if _, ok := f.tt.Find(i18n.T(i18n.KeySidebarExpand)); ok {
		t.Fatal("phone navigation opener remains over fullscreen dock")
	}
	if f.w.Fullscreen != "" {
		t.Fatalf("auto fullscreen persisted mode %q", f.w.Fullscreen)
	}
}

func TestWorkspaceAutoFullscreenOrdinaryOpenExitCloseAndReopen(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, height := range []int{400, 900} {
			t.Run(fmt.Sprintf("external=%v/600x%d", external, height), func(t *testing.T) {
				f := newAutoFullscreenFrame(t, external, 600, height)
				id := f.w.Active
				if f.body.W != 0 {
					t.Fatal("closed resource body mounted")
				}
				if err := f.tt.Click("Toggle rightbar"); err != nil {
					t.Fatal(err)
				}
				requireAutoFrame(t, f, 600)
				// Exit means collapsed + push, and retains the occurrence for reopening.
				if err := f.tt.Click("Restore pane"); err != nil {
					t.Fatal(err)
				}
				if f.open || f.closes != 1 || f.body.W != 0 || f.w.Fullscreen != "" || f.w.Root.find(id) == nil {
					t.Fatal("auto exit did not collapse with retained occurrence")
				}
				if err := f.tt.Click("Toggle rightbar"); err != nil {
					t.Fatal(err)
				}
				requireAutoFrame(t, f, 600)
				// The normal rightbar toggle also closes auto without saving fullscreen.
				if err := f.tt.Click("Toggle rightbar"); err != nil {
					t.Fatal(err)
				}
				if f.open || f.body.W != 0 || f.w.Fullscreen != "" {
					t.Fatal("rightbar collapse left auto body/mode")
				}
				if err := f.tt.Click("Toggle rightbar"); err != nil {
					t.Fatal(err)
				}
				requireAutoFrame(t, f, 600)
				// Closing the final resource still collapses; a later ordinary opening
				// receives the same derived phone presentation.
				if err := f.tt.Click("Close tab"); err != nil {
					t.Fatal(err)
				}
				if f.open || f.body.W != 0 {
					t.Fatal("closing last resource did not collapse")
				}
				f.route = "custom-panels"
				f.open = true
				f.tt.Frame()
				requireAutoFrame(t, f, 600)
				if f.sizes.Right != 540 || f.sizes.Sidebar != SidebarDefault {
					t.Fatal("auto actions changed saved width preferences")
				}
			})
		}
	}
}

func TestWorkspaceAutoFullscreenResizeAndManualMode(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(fmt.Sprintf("external=%v", external), func(t *testing.T) {
			f := newAutoFullscreenFrame(t, external, 600, 400)
			f.open = true
			f.tt.Frame()
			requireAutoFrame(t, f, 600)
			id := f.w.Active
			// Both sides of the exact 768 boundary are exercised on the same owner.
			f.tt.SetSize(768, 400)
			if f.w.Fullscreen != "" || f.body.W <= 0 || f.body.W >= 768 {
				t.Fatalf("auto did not release at 768: mode=%q body=%+v", f.w.Fullscreen, f.body)
			}
			if _, ok := f.tt.Find("Maximize pane"); !ok {
				t.Fatal("desktop dock kept exit glyph")
			}
			f.tt.SetSize(767, 400)
			requireAutoFrame(t, f, 767)
			f.tt.SetSize(1200, 900)
			if f.w.Fullscreen != "" || f.w.Active != id || f.body.W >= 1200 {
				t.Fatal("resizing changed saved mode or occurrence")
			}
			if err := f.tt.Click("Maximize pane"); err != nil {
				t.Fatal(err)
			}
			if f.w.Fullscreen != id {
				t.Fatal("desktop explicit fullscreen was not stored")
			}
			f.tt.SetSize(600, 900)
			if f.w.Fullscreen != id || f.body.X != 0 || f.body.W != 600 {
				t.Fatal("phone observation replaced explicit mode or clipped dock")
			}
			f.tt.SetSize(1200, 900)
			if f.w.Fullscreen != id {
				t.Fatal("phone exit discarded user explicit fullscreen")
			}
			// A close/open toggle retains manual mode. The explicit restore action
			// at phone width intentionally resets push, matching the installed UI.
			if err := f.tt.Click("Toggle rightbar"); err != nil {
				t.Fatal(err)
			}
			if f.open || f.w.Fullscreen != id {
				t.Fatal("ordinary collapse changed manual mode")
			}
			if err := f.tt.Click("Toggle rightbar"); err != nil {
				t.Fatal(err)
			}
			f.tt.SetSize(600, 400)
			if err := f.tt.Click("Restore pane"); err != nil {
				t.Fatal(err)
			}
			if f.open || f.w.Fullscreen != "" {
				t.Fatal("auto restore did not collapse and choose push")
			}
		})
	}
}

func TestWorkspaceAutoFullscreenRetainsSplitFloatAndSavedMode(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprintf("manual=%v", manual), func(t *testing.T) {
			f := newAutoFullscreenFrame(t, true, 1200, 900)
			left := f.w.Root
			right := f.w.leaf("custom-panels")
			floating := f.w.leaf("files")
			floating.Float = &WorkspaceFloat{X: 900, Y: 700, Width: 440, Height: 300, Minimized: true, Order: 2}
			floating.Files = &WorkspaceFileState{Selected: "draft.txt", Preview: "saved", Draft: "unsaved", Editing: true}
			setDockTabs(left, []*WorkspacePane{left, floating})
			root := &WorkspacePane{ID: "retained-split", Axis: "horizontal", Ratio: .37, First: left, Second: right}
			f.w.Root = root
			f.w.Active = left.ID
			f.w.route = f.route
			f.open = true
			if manual {
				f.w.Fullscreen = left.ID
			}
			mode, geometry := f.w.Fullscreen, *floating.Float
			for _, size := range [][2]int{{600, 400}, {600, 900}, {1200, 900}, {767, 400}, {768, 900}} {
				f.tt.SetSize(size[0], size[1])
				if f.w.Root != root || root.First != left || root.Second != right || root.Ratio != .37 || f.w.Root.find(floating.ID) != floating || *floating.Float != geometry || floating.Files.Draft != "unsaved" || f.w.Fullscreen != mode {
					t.Fatalf("resize %v changed retained dock/float/mode", size)
				}
				for _, p := range []*WorkspacePane{left, right} {
					if p.paneWidth <= 0 {
						t.Fatalf("resize %v hid split occurrence %s", size, p.ID)
					}
				}
			}
			raw := f.w.Export()
			var saved struct {
				Fullscreen string `json:"fullscreen"`
			}
			if err := json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.Fullscreen != mode || bytes.Contains(raw, []byte("autoFullscreen")) {
				t.Fatal("derived fullscreen leaked into exported layout")
			}
		})
	}
}

func TestWorkspaceAutoFullscreenNavigationShellKeepsGlobalPageNavigation(t *testing.T) {
	for _, height := range []int{400, 900} {
		t.Run(fmt.Sprintf("600x%d", height), func(t *testing.T) {
			previous := i18n.Locale()
			i18n.SetLocale(i18n.LocaleEN)
			defer i18n.SetLocale(previous)
			sizes := DefaultLayout()
			store := session.New()
			var body ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				NavigationShell(c, Model{Store: store, Sizes: &sizes, RightPaneOpen: true, WorkspacePage: true}, func() { ui.Box(c).Fill().Label("Settings page body").Draw(func(_ *ui.Painter, r ui.Rect) { body = r }) })
			}, 600, height)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			if body.W <= 0 || !tt.HasText("Settings page body") {
				t.Fatal("global settings content disappeared")
			}
			if height == 400 {
				if body.X <= 0 {
					t.Fatal("global landscape navigation rail was bypassed")
				}
			} else if _, ok := tt.Find(i18n.T(i18n.KeySidebarExpand)); !ok {
				t.Fatal("global portrait navigation drawer opener disappeared")
			}
		})
	}
}

func TestWorkspaceAutoFullscreenActualSettingsOverlayPreservesWorkspace(t *testing.T) {
	for _, height := range []int{400, 900} {
		t.Run(fmt.Sprintf("600x%d", height), func(t *testing.T) {
			f := newAutoFullscreenFrame(t, true, 600, height)
			f.open = true
			f.tt.Frame()
			requireAutoFrame(t, f, 600)
			id := f.w.Active
			f.settingsOpen = true
			f.tt.Frame()
			if f.settingsBody.W <= 0 || f.settingsBody.H <= 0 || f.settingsBody.X < 24 || f.settingsBody.X+f.settingsBody.W > 576 || !f.open || f.w.Fullscreen != "" {
				t.Fatalf("actual settings overlay failed body=%+v", f.settingsBody)
			}
			if err := f.tt.Click("Toggle rightbar"); err != nil {
				t.Fatal(err)
			}
			if !f.open || f.w.Fullscreen != "" {
				t.Fatal("foreground settings allowed background rightbar action")
			}
			// In portrait the toolbar is outside the centered modal: the
			// backdrop may dismiss settings, but cannot activate that toolbar.
			f.settingsOpen = true
			f.tt.Frame()
			f.tt.Key(0, ui.KeyEscape)
			if f.settingsOpen || f.w.Active != id {
				t.Fatal("settings dismissal changed resource occurrence")
			}
			requireAutoFrame(t, f, 600)
		})
	}
}

func TestWorkspaceAutoFullscreenComposerDraftSelectionAndTranscriptSurviveBoundary(t *testing.T) {
	f := newAutoFullscreenFrame(t, true, 768, 900)
	f.store.Select("auto-draft-session")
	for i := 0; i < 32; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		f.store.AppendStreamDelta(role, fmt.Sprintf("Retained history row %d: %s", i, strings.Repeat("conversation history ", 8)))
	}
	f.draft = "ABCDE"
	f.client.Sync()
	f.open = true
	f.tt.Frame()
	f.chatList.ScrollTo(10, ui.Start)
	f.tt.Frame()
	first, _ := f.chatList.Visible()
	if first == 0 {
		t.Fatal("fixture did not scroll transcript away from start")
	}
	if err := f.tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	f.client.SetSelectionUTF16(1, 4)
	f.tt.Frame()
	retained := f.tt.NativeTextInputClient()
	if retained == nil {
		t.Fatal("fixture native draft did not focus")
	}
	f.tt.SetSize(767, 900)
	if f.tt.NativeTextInputClient() != nil || f.tt.Focused("Composer draft") {
		t.Fatal("hidden composer retained native keyboard authority")
	}
	// A stale OS adapter must fail closed while its surface is unmounted.
	retained.ReplaceText(nil, "stale")
	if f.draft != "ABCDE" || f.client.Selection().Range != (ui.TextInputRange{Start: 1, End: 4}) {
		t.Fatal("phone presentation lost draft/selection or accepted stale input")
	}
	f.tt.SetSize(768, 900)
	if f.draft != "ABCDE" || f.client.Selection().Range != (ui.TextInputRange{Start: 1, End: 4}) {
		t.Fatal("returning to dock changed retained draft selection")
	}
	after, _ := f.chatList.Visible()
	if after != first {
		t.Fatalf("boundary lost transcript anchor %d -> %d", first, after)
	}
	// Explicit refocus uses the retained document/selection, never the stale
	// disposed native adapter, and editing resumes at the same selected range.
	if err := f.tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	f.client.SetSelectionUTF16(1, 4)
	f.tt.Frame()
	f.tt.Type("Z")
	if f.draft != "AZE" {
		t.Fatalf("retained composer could not edit after resize: %q", f.draft)
	}
}
