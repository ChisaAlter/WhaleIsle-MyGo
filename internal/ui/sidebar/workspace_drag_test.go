package sidebar

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestWorkspaceDropPlacementUsesWholeParentSection(t *testing.T) {
	groups := []session.WorkspaceGroup{{Key: "parent", Kind: "workspace"}, {Key: "child", Kind: "workspace", ParentID: "parent", Depth: 1}, {Key: "grandchild", Kind: "workspace", ParentID: "child", Depth: 2}, {Key: "last", Kind: "workspace"}}
	bounds := []ui.Rect{{X: 0, Y: 0, W: 280, H: 68}, {X: 0, Y: 72, W: 280, H: 68}, {X: 0, Y: 144, W: 280, H: 34}, {X: 0, Y: 182, W: 280, H: 34}}
	for _, tc := range []struct {
		name, source string
		x, y         float32
		target, edge int
		after        bool
		before       string
		ok           bool
	}{
		{"parent-before", "last", 20, 2, 0, 0, false, "parent", true},
		{"parent-after-at-descendant", "last", 20, 175, 0, 2, true, "", false},
		{"move-parent-after-last", "parent", 20, 215, 3, 3, true, "", true},
		{"child-cannot-cross-parent", "child", 20, 200, 0, 0, false, "", false},
		{"outside", "last", -2, 10, 0, 0, false, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, edge, after, before, ok := workspaceDropPlacement(groups, bounds, tc.source, tc.x, tc.y)
			if ok != tc.ok || (ok && (target != tc.target || edge != tc.edge || after != tc.after || before != tc.before)) {
				t.Fatalf("target=%d edge=%d after=%v before=%q ok=%v", target, edge, after, before, ok)
			}
		})
	}
	// Put a third sibling after last: dropping it at the parent's last
	// descendant must place it before last and draw at that descendant's edge.
	groups = append(groups, session.WorkspaceGroup{Key: "source", Kind: "workspace"})
	bounds = append(bounds, ui.Rect{X: 0, Y: 220, W: 280, H: 34})
	target, edge, after, before, ok := workspaceDropPlacement(groups, bounds, "source", 20, 175)
	if !ok || target != 0 || edge != 2 || !after || before != "last" {
		t.Fatalf("parent end target=%d edge=%d after=%v before=%q ok=%v", target, edge, after, before, ok)
	}
}

func drawDragWorkspaceFixture(c *ui.Context, m Model, groups []session.WorkspaceGroup, collapsed map[string]bool) {
	list := ui.Column(c.Key("drag-list")).FillWidth().Gap(4)
	m = workspaceDragModel(list, m)
	blocks := []workspaceDropBlock{}
	list.Children(func() {
		hidden := map[string]bool{}
		for _, g := range groups {
			if hidden[g.ParentID] {
				hidden[g.Key] = true
				continue
			}
			block := ui.Column(c.Key(workspaceRowKey{m.Snapshot.WorkspaceLayout.HostGeneration, g.Key})).FillWidth().Gap(2).Shrink(0).Label("Block " + g.Key)
			blocks = append(blocks, workspaceDropBlock{g, block})
			block.Children(func() {
				header := ui.Row(c).Height(34).FillWidth().Label("Header " + g.Key)
				header.Children(func() {
					btn := ui.Button(c, g.Key).Height(34).Grow(1)
					workspaceDragSource(btn, m, g.Key)
					if btn.Clicked() && m.OnToggleWorkspaceGroup != nil {
						m.OnToggleWorkspaceGroup(g.Key)
					}
					ui.Button(c, "…").Size(20, 20).Label("Menu " + g.Key).Menu(func(menu *ui.Menu) {
						if menu.Item("Manage "+g.Key).Chosen() && m.OnManageWorkspace != nil {
							m.OnManageWorkspace(g.Key)
						}
					})
				})
				if !collapsed[g.Key] {
					ui.Box(c).Height(32).FillWidth().Label("Session " + g.Key)
				}
			})
			if collapsed[g.Key] {
				hidden[g.Key] = true
			}
		}
		workspaceDropGroupsTarget(list, m, blocks, ui.RGB(250, 10, 20))
	})
}

func TestWorkspaceDropNoLayoutSlotsAndMenuPriority(t *testing.T) {
	groups := []session.WorkspaceGroup{{Key: "parent", Kind: "workspace"}, {Key: "child", Kind: "workspace", ParentID: "parent", Depth: 1}, {Key: "last", Kind: "workspace"}, {Key: "source", Kind: "workspace"}}
	var moves, toggles, managed []string
	m := Model{Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 7}}, OnDropWorkspaceBefore: func(g uint64, id, before string) {
		if g != 7 {
			t.Error("generation")
		}
		moves = append(moves, id+"/"+before)
	}, OnToggleWorkspaceGroup: func(id string) { toggles = append(toggles, id) }, OnManageWorkspace: func(id string) { managed = append(managed, id) }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { drawDragWorkspaceFixture(c, m, groups, nil) })
	}, 280, 440)
	parent, _ := tt.Find("Block parent")
	child, _ := tt.Find("Block child")
	last, _ := tt.Find("Block last")
	if parent.H != 68 || child.Y != parent.Y+parent.H+4 || last.Y != child.Y+child.H+4 {
		t.Fatalf("layout blocks parent=%+v child=%+v last=%+v", parent, child, last)
	}
	if err := tt.Click("Menu parent"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Manage parent"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(managed, []string{"parent"}) || len(toggles) != 0 || len(moves) != 0 {
		t.Fatalf("menu leaked managed=%v toggles=%v moves=%v", managed, toggles, moves)
	}
	src, _ := tt.Find("source")
	tt.Press(src.X+10, src.Y+10)
	tt.Move(src.X+20, src.Y+10)
	tt.Move(child.X+10, child.Y+child.H-3)
	p2, _ := tt.Find("Block parent")
	c2, _ := tt.Find("Block child")
	l2, _ := tt.Find("Block last")
	if p2 != parent || c2 != child || l2 != last {
		t.Fatal("drag marker changed row geometry")
	}
	// Inspect the actual paint: the after marker is at child bottom, not
	// parent header bottom, and still has no layout slot.
	img := tt.Image()
	pixel := img.RGBAAt(int(child.X+child.W-2), int(child.Y+child.H-1))
	if pixel.R < 200 || pixel.G > 80 {
		t.Fatalf("parent-end marker missing: %v", pixel)
	}
	tt.Release(child.X+10, child.Y+child.H-3)
	if !reflect.DeepEqual(moves, []string{"source/last"}) || len(toggles) != 0 {
		t.Fatalf("moves=%v toggles=%v", moves, toggles)
	}
	// Reverse to the upper half and retain Escape cancellation.
	tt.Press(src.X+10, src.Y+10)
	tt.Move(parent.X+20, parent.Y+3)
	tt.Key(0, ui.KeyEscape)
	tt.Release(parent.X+20, parent.Y+3)
	if len(moves) != 1 {
		t.Fatal("Escape committed")
	}
}

func TestWorkspaceDropCollapsedAndHostGeneration(t *testing.T) {
	groups := []session.WorkspaceGroup{{Key: "a", Kind: "workspace"}, {Key: "child", Kind: "workspace", ParentID: "a", Depth: 1}, {Key: "b", Kind: "workspace"}, {Key: "c", Kind: "workspace"}}
	var moves []string
	m := Model{Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 11}}, OnDropWorkspaceBefore: func(_ uint64, id, before string) { moves = append(moves, id+"/"+before) }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { drawDragWorkspaceFixture(c, m, groups, map[string]bool{"a": true}) })
	}, 280, 300)
	a, _ := tt.Find("Block a")
	b, _ := tt.Find("Block b")
	if a.H != 34 || b.Y != a.Y+a.H+4 || tt.HasText("child") {
		t.Fatalf("collapsed geometry a=%+v b=%+v", a, b)
	}
	src, _ := tt.Find("c")
	tt.Press(src.X+10, src.Y+10)
	tt.Move(a.X+10, a.Y+a.H-2)
	tt.Release(a.X+10, a.Y+a.H-2)
	if !reflect.DeepEqual(moves, []string{"c/b"}) {
		t.Fatalf("collapsed drop %v", moves)
	}
	// A host change during the drag must not move the replacement host.
	tt.Press(src.X+10, src.Y+10)
	tt.Move(a.X+10, a.Y+2)
	m.Snapshot.WorkspaceLayout.HostGeneration = 12
	tt.Frame()
	tt.Release(a.X+10, a.Y+2)
	if len(moves) != 1 {
		t.Fatalf("old Host drag committed %v", moves)
	}
}
