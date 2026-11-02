package sidebar

import (
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceProductDropUsesVisibleTreeEndAndKeepsActions(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	items := []session.Item{{SessionID: "a", Title: "Parent session"}, {SessionID: "b", Title: "Child session"}}
	var moves, selected, created, toggled, managed []string
	m := Model{Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "p", Title: "Parent", Path: "/parent", SessionIDs: []string{"a"}}, {WorkspaceID: "ch", Title: "Child", Path: "/parent/child", SessionIDs: []string{"b"}}, {WorkspaceID: "last", Title: "Last", Path: "/last"}, {WorkspaceID: "src", Title: "Source", Path: "/source"}}, WorkspaceLayout: session.WorkspaceLayoutState{GroupBy: "workspace-tree", HostGeneration: 15, Collapsed: map[string]bool{"p": false, "ch": false, "last": false, "src": false}}}, OnDropWorkspaceBefore: func(g uint64, id, before string) {
		if g != 15 {
			t.Error("wrong Host")
		}
		moves = append(moves, id+"/"+before)
	}, OnSelect: func(id string) { selected = append(selected, id) }, OnCreateWorkspaceSession: func(g uint64, id string) {
		if g != 15 {
			t.Error("wrong create Host")
		}
		created = append(created, id)
	}, OnBeginWorkspaceRename: func(id string) { managed = append(managed, id) }}
	m.OnToggleWorkspaceGroup = func(id string) {
		toggled = append(toggled, id)
		m.Snapshot.WorkspaceLayout.Collapsed[id] = !m.Snapshot.WorkspaceLayout.Collapsed[id]
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(250, 10, 20), ui.RGB(90, 90, 90), ui.Transparent) })
	}, 300, 360)
	// Source is a root workspace. Crossing its parent's descendant rows must
	// target the complete parent, never try to reparent Source under Child.
	src, _ := tt.Find("Source")
	child, _ := tt.Find("Child session")
	parent, _ := tt.Find("Parent")
	last, _ := tt.Find("Last")
	tt.Press(src.X+8, src.Y+src.H/2)
	tt.Move(src.X+16, src.Y+src.H/2)
	tt.Move(child.X+5, child.Y+child.H-2)
	p2, _ := tt.Find("Parent")
	ch2, _ := tt.Find("Child session")
	l2, _ := tt.Find("Last")
	if parent != p2 || child != ch2 || last != l2 {
		t.Fatal("product marker consumed vertical space")
	}
	// The ghost header is only 32px high; its copy cannot cover this far edge.
	pixel := tt.Image().RGBAAt(298, int(child.Y+child.H-1))
	if pixel.R < 200 || pixel.G > 80 {
		t.Fatalf("marker not at parent group's last row: %v", pixel)
	}
	tt.Release(child.X+5, child.Y+child.H-2)
	if !reflect.DeepEqual(moves, []string{"src/last"}) || len(toggled) != 0 || len(selected) != 0 {
		t.Fatalf("parent-end moves=%v toggled=%v selected=%v", moves, toggled, selected)
	}
	// The same root's upper boundary and the overall last root's lower half.
	sidebarDragBetween(tt, "Source", "Parent", false)
	sidebarDragBetween(tt, "Parent", "Source", true)
	if !reflect.DeepEqual(moves, []string{"src/last", "src/p", "p/"}) {
		t.Fatalf("before/append %v", moves)
	}
	// Exact real hover actions remain independent from the draggable title.
	if err := tt.Click(workspaceText("工作区操作 ", "Workspace actions ") + "p"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(workspaceText("重命名", "Rename")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(workspaceText("新建会话 ", "New session ") + "p"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Parent session"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(managed, []string{"p"}) || !reflect.DeepEqual(created, []string{"p"}) || !reflect.DeepEqual(selected, []string{"a"}) || len(toggled) != 0 || len(moves) != 3 {
		t.Fatalf("actions managed=%v created=%v selected=%v toggled=%v moves=%v", managed, created, selected, toggled, moves)
	}
	now := time.Now()
	tt.SetTime(now)
	if err := tt.Click("Parent"); err != nil {
		t.Fatal(err)
	}
	// Logical folding retires descendants and outgoing rows from semantic/input
	// lookup immediately; the outgoing parent's pixels still fade independently.
	if tt.HasText("Child") || tt.HasText("Child session") || tt.HasText("Parent session") {
		t.Fatal("fold retained semantic/input authority for hidden or outgoing rows")
	}
	if err := tt.Click("Parent session"); err == nil {
		t.Fatal("inert outgoing row remained addressable")
	}
	if !reflect.DeepEqual(selected, []string{"a"}) {
		t.Fatal("outgoing session remained selectable")
	}
	tt.SetTime(now.Add(210 * time.Millisecond))
	if tt.HasText("Child") || tt.HasText("Child session") || tt.HasText("Parent session") {
		t.Fatal("closing parent retained hidden rows")
	}
	p3, _ := tt.Find("Parent")
	l3, _ := tt.Find("Last")
	if l3.Y-p3.Y != 38 {
		t.Fatalf("collapsed product pitch %v", l3.Y-p3.Y)
	}
	sidebarDragBetween(tt, "Source", "Parent", true)
	if moves[len(moves)-1] != "src/last" || len(moves) != 4 {
		t.Fatalf("closed group after %v", moves)
	}
}
