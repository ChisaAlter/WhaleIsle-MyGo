package frame

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestRightDividerKeyboardAndNoOpPress(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	sizes.Right = 540
	right := float32(0)
	tt := ui.NewTester(func(c *ui.Context) {
		cols := sizes.ResolveColumns(1200, true)
		right = cols.Details
		centerRightColumns(c, Model{Sizes: &sizes, FilePaneWidth: &right}, func() {}, func() {})
	}, 1200, 800)
	// Isolated center/right row starts at0 here, so the handle is at400.
	tt.Press(400, 300)
	tt.Release(400, 300)
	if sizes.Right != 540 {
		t.Fatalf("no-op press changed remembered width to%v", sizes.Right)
	}
	tt.Key(0, ui.KeyRight)
	if sizes.Right != 508 || right != 508 || sizes.Center != 410 {
		t.Fatalf("keyboard resize sizes=%+v right=%v", sizes, right)
	}
}

// The callbacks read rectangles supplied by the real software painter. These
// are actual laid-out pane boxes, rather than assertions on the solver alone.
func TestShellColumnGeometryPaintBounds(t *testing.T) {
	type bounds struct {
		Width                            int
		Sidebar, Center, Right, Terminal ui.Rect
	}
	var evidence []bounds
	for _, width := range []int{900, 1200, 1600} {
		sizes := DefaultLayout()
		sizes.Right = 540
		if width == 1600 {
			sizes.Right = 720
		}
		right := float32(0)
		got := bounds{Width: width}
		tt := ui.NewTester(func(c *ui.Context) {
			cols := sizes.ResolveColumns(float32(width), true)
			right = cols.Details
			sidebarColumns(c, &sizes, func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { got.Sidebar = r }) }, func() {
				centerRightColumns(c, Model{Sizes: &sizes, FilePaneWidth: &right}, func() {
					ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { got.Center = r }).Children(func() {
						centerTerminalColumn(c, Model{Sizes: &sizes, centerHeight: 800}, func() { ui.Box(c).Fill() }, func() {
							ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { got.Terminal = r })
						})
					})
				}, func() { ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) { got.Right = r }) })
			})
		}, width, 800)
		_ = tt
		rail := float32(280)
		if width < 1024 {
			rail = SidebarCollapsedWidth()
		}
		sidebarDivider := float32(1)
		if rail == 0 {
			sidebarDivider = 0
		}
		wantCenter := ui.Rect{X: rail + sidebarDivider, Y: 0, W: sizes.Center, H: 800}
		wantSidebar := ui.Rect{W: rail, H: 800}
		if rail == 0 {
			wantSidebar = ui.Rect{}
		}
		if got.Sidebar != wantSidebar || got.Center != wantCenter {
			t.Fatalf("width%d painted bounds=%+v", width, got)
		}
		if got.Terminal != (ui.Rect{X: rail + sidebarDivider, Y: 561, W: sizes.Center, H: 239}) {
			t.Fatalf("width%d terminal=%+v", width, got.Terminal)
		}
		if right > 0 && got.Right != (ui.Rect{X: rail + sidebarDivider + 1 + sizes.Center, Y: 0, W: right, H: 800}) {
			t.Fatalf("width%d right=%+v", width, got.Right)
		}
		if right == 0 && got.Right != (ui.Rect{}) {
			t.Fatalf("width%d hidden right=%+v", width, got.Right)
		}
		evidence = append(evidence, got)
	}
	if dir := os.Getenv("SHELL_SNAPSHOT_DIR"); dir != "" {
		data, err := json.MarshalIndent(evidence, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, "columns-drawn-bounds.json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTerminalDividerFrozenExcursionAndKeyboard(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Right = 540
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true}) }, 1200, 800)
	tt.Press(400, 560)
	tt.Release(400, 560)
	if sizes.Main != 560 {
		t.Fatalf("no-op terminal press changed Main%v", sizes.Main)
	}
	tt.Key(0, ui.KeyUp)
	if sizes.Main != 550 {
		t.Fatalf("terminal keyboard Main%v, want550", sizes.Main)
	}
	tt.Press(400, 550)
	for _, y := range []float32{700, 780, 740, 700, 550} {
		tt.Move(400, y)
	}
	tt.Release(400, 550)
	if sizes.Main != 550 {
		t.Fatalf("round-trip terminal drag Main%v, want frozen550", sizes.Main)
	}
}
