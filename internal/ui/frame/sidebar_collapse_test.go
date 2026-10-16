package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/recovery"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSidebarCollapseRestore(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Sidebar = 336
	sizes.Bottom = 0
	st := session.New()
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: st, Sizes: &sizes}) }, 1200, 800)
	// This state/command regression uses the native reduced-motion contract;
	// shell_motion_test covers the actual transition and native bounds.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	// Real pointer resizing, then both command and visible-button toggles.
	tt.Press(336, 300)
	tt.Move(360, 300)
	tt.Release(360, 300)
	if sizes.Sidebar != 360 {
		t.Fatalf("drag width=%v", sizes.Sidebar)
	}
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	if sizes.Sidebar != 0 || sizes.SidebarExpanded != 360 {
		t.Fatalf("collapse lost preference: %+v", sizes)
	}
	if _, ok := tt.Find("Sidebar divider"); ok {
		t.Fatal("collapsed rail kept a resize target")
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarExpand)); err != nil {
		t.Fatal(err)
	}
	if sizes.Sidebar != 360 {
		t.Fatalf("expanded width lost: %v", sizes.Sidebar)
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarCollapse)); err != nil {
		t.Fatal(err)
	}
	if sizes.Sidebar != 0 {
		t.Fatal("collapse button failed")
	}
	tt.Key(ui.Cmd, ui.KeyB)
	if sizes.Sidebar != 360 {
		t.Fatal("keyboard restore failed")
	}
	// Compact painted geometry, not just the solver's return value.
	sizes.ToggleSidebar()
	var rail, main ui.Rect
	geometry := ui.NewTester(func(c *ui.Context) {
		sidebarColumns(c, &sizes,
			func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { rail = r }) },
			func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { main = r }) })
	}, 1200, 800)
	_ = geometry
	if rail.W != SidebarCollapsed || main.X != SidebarCollapsed+SplitterWidth {
		t.Fatalf("rail=%+v main=%+v", rail, main)
	}
	// Existing resume store round-trips closed state and the original width.
	dir := t.TempDir()
	saved, err := recovery.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = saved.Update(func(s *recovery.State) {
		s.Layout = recovery.Layout{Sidebar: sizes.Sidebar, SidebarExpanded: sizes.SidebarExpanded}
	}); err != nil {
		t.Fatal(err)
	}
	reopened, err := recovery.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	l := reopened.Snapshot().Layout
	resumed := LayoutSizes{Sidebar: l.Sidebar, SidebarExpanded: l.SidebarExpanded}
	resumed.ToggleSidebar()
	if resumed.Sidebar != 360 {
		t.Fatalf("resume width=%v", resumed.Sidebar)
	}
	old := LayoutSizes{}
	old.ToggleSidebar()
	if old.Sidebar != SidebarDefault {
		t.Fatal("missing old width did not use default")
	}
	// Same page-command dispatch guard as all other commands: editable/IME and
	// custom input (terminal/native guest owners) retain Ctrl+B.
	mode, text, custom := "editor", "", 0
	owned := ui.NewTester(func(c *ui.Context) {
		registerPageCommands(c, Model{Store: st, Sizes: &resumed})
		if mode == "editor" {
			ui.TextInput(c, &text).Label("Editor").Focus()
		} else {
			ui.Column(c).Focusable().Focus().HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputKeyDown {
					custom++
					return true
				}
				return false
			})
		}
	}, 600, 400)
	owned.Key(ui.Cmd, ui.KeyB)
	owned.Compose("ni", 2)
	owned.Key(ui.Cmd, ui.KeyB)
	owned.Type("你")
	if resumed.Sidebar != 360 || text != "你" {
		t.Fatal("text/IME command stolen")
	}
	mode = "custom"
	owned.Frame()
	owned.Key(ui.Cmd, ui.KeyB)
	if resumed.Sidebar != 360 || custom != 1 {
		t.Fatal("custom input command stolen")
	}
}
