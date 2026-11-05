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
	tt.PhysicalKeyDown("KeyB", ui.KeyB, ui.Cmd, false, false, false)
	tt.PhysicalKeyUp("KeyB", ui.KeyB, ui.Cmd, false)
	tt.Frame()
	if sizes.Sidebar != 0 || sizes.SidebarExpanded != 360 {
		t.Fatalf("collapse lost preference: %+v", sizes)
	}
	if _, ok := tt.Find("Sidebar divider"); ok {
		t.Fatal("collapsed rail kept a resize target")
	}
	if SidebarCollapsedWidth() == 0 {
		sizes.ToggleSidebar()
		tt.Frame()
	} else {
		if err := tt.Click(i18n.T(i18n.KeySidebarExpand)); err != nil {
			t.Fatal(err)
		}
	}
	if sizes.Sidebar != SidebarDefault {
		t.Fatalf("wide reopen did not use source default: %v", sizes.Sidebar)
	}
	if SidebarCollapsedWidth() == 0 {
		sizes.ToggleSidebar()
		tt.Frame()
	} else {
		if err := tt.Click(i18n.T(i18n.KeySidebarCollapse)); err != nil {
			t.Fatal(err)
		}
	}
	if sizes.Sidebar != 0 {
		t.Fatal("collapse button failed")
	}
	tt.PhysicalKeyDown("KeyB", ui.KeyB, ui.Cmd, false, false, false)
	tt.PhysicalKeyUp("KeyB", ui.KeyB, ui.Cmd, false)
	if sizes.Sidebar != SidebarDefault {
		t.Fatal("keyboard restore did not use source default")
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
	collapsed := SidebarCollapsedWidth()
	divider := SplitterWidth
	if collapsed == 0 {
		divider = 0
	}
	if rail.W != collapsed || main.X != collapsed+divider {
		t.Fatalf("rail=%+v main=%+v", rail, main)
	}
	// Resume retains closed state; explicit wide reopen uses the source default.
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
	if resumed.Sidebar != SidebarDefault {
		t.Fatalf("resume width=%v", resumed.Sidebar)
	}
	old := LayoutSizes{}
	old.ToggleSidebar()
	if old.Sidebar != SidebarDefault {
		t.Fatal("missing old width did not use default")
	}
	// Installed native-priority commands work from editable/custom owners;
	// composition itself still owns its committing key.
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
	owned.PhysicalKeyDown("KeyB", ui.KeyB, ui.Cmd, false, false, false)
	owned.PhysicalKeyUp("KeyB", ui.KeyB, ui.Cmd, false)
	owned.Compose("ni", 2)
	owned.PhysicalKeyDown("KeyB", ui.KeyB, ui.Cmd, false, false, false)
	owned.PhysicalKeyUp("KeyB", ui.KeyB, ui.Cmd, false)
	owned.Type("你")
	if resumed.Sidebar != 0 || text != "你" {
		t.Fatal("text/IME command stolen")
	}
	mode = "custom"
	owned.Frame()
	owned.PhysicalKeyDown("KeyB", ui.KeyB, ui.Cmd, false, false, false)
	owned.PhysicalKeyUp("KeyB", ui.KeyB, ui.Cmd, false)
	if resumed.Sidebar != SidebarDefault || custom != 0 {
		t.Fatal("custom input command stolen")
	}
}
