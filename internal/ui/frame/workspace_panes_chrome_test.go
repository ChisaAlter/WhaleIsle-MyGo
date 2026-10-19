package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func paneChromeFixture(width int) (*ui.Tester, *WorkspacePanes, *LayoutSizes) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	sizes := DefaultLayout()
	sizes.Workspace = w
	tt := ui.NewTester(func(c *ui.Context) {
		defer func() {
			if w.pending != nil {
				action := w.pending
				w.pending = nil
				action()
			}
		}()
		n := w.Root.dockedLeaf()
		if n == nil {
			return
		}
		m := Model{Sizes: &sizes, workspacePane: n, RightPaneOpen: true}
		surface := ui.Column(c).Fill()
		surface.Children(func() {
			workspacePaneControls(c, m)
			ui.Button(c, "").Label("Pane body").FillWidth().Grow(1)
			workspaceDockDropTargets(c, m, surface)
		})
	}, width, 400)
	return tt, w, &sizes
}

func TestWorkspacePaneChromeHeightAndNarrowActions(t *testing.T) {
	for _, width := range []int{300, 420, 800} {
		tt, _, _ := paneChromeFixture(width)
		header, ok := tt.Find("Workspace pane header")
		if !ok || header.H != 38 {
			t.Fatalf("width=%d header=%+v", width, header)
		}
		body, _ := tt.Find("Pane body")
		if body.Y != 38 {
			t.Fatalf("width=%d body starts at %v, want 38", width, body.Y)
		}
		for _, label := range []string{paneLabel("面板操作", "Pane actions"), paneLabel("面板全屏", "Maximize pane"), paneLabel("关闭标签", "Close tab")} {
			r, ok := tt.Find(label)
			if !ok || r.X < 0 || r.X+r.W > float32(width) || r.H != 28 {
				t.Fatalf("width=%d action=%q bounds=%+v", width, label, r)
			}
		}
		if _, ok := tt.Find("Dock drop left"); ok {
			t.Fatal("idle pane mounted drag hints")
		}
		if tt.HasText("Active pane") || tt.HasText(paneLabel("重置布局", "Reset layout")) {
			t.Fatal("diagnostic action row remains visible")
		}
	}
}

func TestWorkspacePaneMenuRetainsActionsAndResourceIdentity(t *testing.T) {
	tt, w, _ := paneChromeFixture(300)
	original := w.Root
	resourceID := original.ID
	filesState := &WorkspaceFileState{}
	original.Files = filesState
	if err := tt.Click(paneLabel("面板操作", "Pane actions")); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(paneLabel("向右拆分", "Split right")); err != nil {
		t.Fatal(err)
	}
	if w.Root.First == nil || w.Root.First.ID != resourceID || w.Root.First.Files != filesState || w.Root.Second == nil {
		t.Fatal("menu split lost original resource")
	}
	// Context menus belong to the tab under the pointer, not the active seat.
	tt, w, _ = paneChromeFixture(420)
	files := w.Root
	w.selectKind(files, "diff")
	tt.Frame()
	if err := tt.RightClick(dockTabTitle(files)); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(paneLabel("浮动", "Float")); err != nil {
		t.Fatal(err)
	}
	if w.Root.find(files.ID) != files || files.Float == nil {
		t.Fatal("context menu floated the wrong resource")
	}
}

func TestWorkspacePaneChromeMaximizeAndClose(t *testing.T) {
	tt, w, _ := paneChromeFixture(300)
	original := w.Root
	if err := tt.Click(paneLabel("面板全屏", "Maximize pane")); err != nil {
		t.Fatal(err)
	}
	if w.Fullscreen != original.ID {
		t.Fatal("maximize glyph did not expand dock")
	}
	if err := tt.Click(paneLabel("退出全屏", "Restore pane")); err != nil {
		t.Fatal(err)
	}
	if w.Fullscreen != "" || w.Root != original {
		t.Fatal("restore glyph replaced resource")
	}
	if err := tt.Click(paneLabel("关闭标签", "Close tab")); err != nil {
		t.Fatal(err)
	}
	if w.Root.find(original.ID) != nil {
		t.Fatal("close glyph did not close its target")
	}
}
