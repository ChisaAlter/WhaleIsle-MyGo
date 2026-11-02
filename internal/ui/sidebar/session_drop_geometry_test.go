package sidebar

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func sidebarDragBetween(tt *ui.Tester, source, target string, after bool) {
	s, _ := tt.Find(source)
	d, _ := tt.Find(target)
	y := d.Y + 3
	if after {
		y = d.Y + d.H - 3
	}
	tt.Press(s.X+5, s.Y+s.H/2)
	tt.Move(s.X+12, s.Y+s.H/2)
	tt.Move(d.X+5, y)
	tt.Release(d.X+5, y)
}

func TestSessionRowDropHalvesMenuAndCompactPitch(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	items := []session.Item{{SessionID: "a", Title: "Alpha"}, {SessionID: "b", Title: "Beta"}, {SessionID: "c", Title: "Gamma"}}
	var moves, selected, pins []string
	m := Model{Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Workspace", SessionIDs: []string{"a", "b", "c"}}}, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 3, Collapsed: map[string]bool{"ws": false}}}, OnMoveWorkspaceSessionBefore: func(w, id, before string) { moves = append(moves, w+"/"+id+"/"+before) }, OnSelect: func(id string) { selected = append(selected, id) }, OnPin: func(id string) { pins = append(pins, id) }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(10, 20, 30), ui.RGB(90, 90, 90), ui.Transparent) })
	}, 300, 260)
	a, _ := tt.Find("Alpha")
	b, _ := tt.Find("Beta")
	g, _ := tt.Find("Gamma")
	if a.H != 32 || b.Y-a.Y != 34 || g.Y-b.Y != 34 {
		t.Fatalf("session pitch a=%+v b=%+v g=%+v", a, b, g)
	}
	if err := tt.Click("Alpha"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(selected, []string{"a"}) || len(moves) != 0 {
		t.Fatalf("click selected=%v moves=%v", selected, moves)
	}
	tt.Move(a.X+10, a.Y+a.H/2)
	if err := tt.Click(sessionActionsLabel("a")); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(sessionPinLabel(false)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pins, []string{"a"}) || len(selected) != 1 || len(moves) != 0 {
		t.Fatalf("menu pins=%v selected=%v moves=%v", pins, selected, moves)
	}
	sidebarDragBetween(tt, "Gamma", "Alpha", true)
	sidebarDragBetween(tt, "Gamma", "Alpha", false)
	sidebarDragBetween(tt, "Alpha", "Gamma", true)
	if !reflect.DeepEqual(moves, []string{"ws/c/b", "ws/c/a", "ws/a/"}) || len(selected) != 1 {
		t.Fatalf("halves moves=%v selected=%v", moves, selected)
	}
	a2, _ := tt.Find("Alpha")
	b2, _ := tt.Find("Beta")
	g2, _ := tt.Find("Gamma")
	if a2 != a || b2 != b || g2 != g {
		t.Fatalf("row geometry changed after drops a=%+v b=%+v g=%+v", a2, b2, g2)
	}
}

func TestSessionRowDropRejectsIncompatibleAccountsAndAllowsOrdinaryFork(t *testing.T) {
	for _, mode := range []string{"workspace", "pinned", "ordinary-fork", "busy", "archived"} {
		t.Run(mode, func(t *testing.T) {
			items := []session.Item{{SessionID: "a", Title: "Alpha"}, {SessionID: "b", Title: "Beta"}, {SessionID: "c", Title: "Gamma"}}
			var moves []string
			m := Model{Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "one", Title: "One", SessionIDs: []string{"a", "b", "c"}}}, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 2, Collapsed: map[string]bool{"one": false, "two": false}}}, OnMoveWorkspaceSessionBefore: func(workspace, id, before string) { moves = append(moves, workspace+"/"+id+"/"+before) }}
			switch mode {
			case "workspace":
				m.Snapshot.Workspaces = []session.WorkspaceRow{{WorkspaceID: "one", Title: "One", SessionIDs: []string{"a", "b"}}, {WorkspaceID: "two", Title: "Two", SessionIDs: []string{"c"}}}
			case "pinned":
				m.Snapshot.PinnedSessionIDs = []string{"c"}
				items[2].Pinned = true
			case "ordinary-fork":
				items[2].ParentID = "b"
			case "busy":
				m.Snapshot.WorkspaceLayout.Reordering = true
			case "archived":
				m.Snapshot.ArchivedSessionIDs = []string{"c"}
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(20, 20, 20), ui.RGB(90, 90, 90), ui.Transparent) })
			}, 300, 300)
			if mode == "ordinary-fork" {
				parent, parentVisible := tt.Find("Beta")
				fork, forkVisible := tt.Find("Gamma")
				if !parentVisible || !forkVisible || parent.X != fork.X {
					t.Fatalf("ordinary fork is not a same-depth row: parent=%+v fork=%+v", parent, fork)
				}
			}
			sidebarDragBetween(tt, "Gamma", "Alpha", false)
			if mode == "ordinary-fork" {
				if !reflect.DeepEqual(moves, []string{"one/c/a"}) {
					t.Fatalf("same-depth ordinary fork move=%v", moves)
				}
			} else if len(moves) != 0 {
				t.Fatalf("incompatible drag committed: %v", moves)
			}
		})
	}
}

func TestCompactWorkspaceGeometryFlatScratchAndTree(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, mode := range []string{"workspace", "flat", "workspace-tree", "collapsed"} {
		t.Run(mode, func(t *testing.T) {
			items := []session.Item{{SessionID: "a", Title: "Parent session"}, {SessionID: "b", Title: "Child session"}, {SessionID: "c", Title: "Last session"}, {SessionID: "s", Title: "Scratch session", Cwd: "/scratch"}}
			m := Model{Snapshot: session.ViewState{Items: items, ScratchCwd: "/scratch", Workspaces: []session.WorkspaceRow{{WorkspaceID: "parent", Title: "Parent", Path: "/one", SessionIDs: []string{"a"}}, {WorkspaceID: "child", Title: "Child", Path: "/one/child", SessionIDs: []string{"b"}}, {WorkspaceID: "last", Title: "Last", Path: "/last", SessionIDs: []string{"c"}}}, WorkspaceLayout: session.WorkspaceLayoutState{GroupBy: mode, HostGeneration: 3, Collapsed: map[string]bool{"parent": false, "child": false, "last": false, "::scratch": false}}}, OnMoveWorkspaceSessionBefore: func(string, string, string) {}, OnMoveLocalSessionBefore: func(uint64, string, string, string) {}}
			if mode == "collapsed" {
				m.Snapshot.WorkspaceLayout.GroupBy = "workspace-tree"
				m.Snapshot.WorkspaceLayout.Collapsed["parent"] = true
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(10, 20, 30), ui.RGB(90, 90, 90), ui.Transparent) })
			}, 300, 480)
			if mode == "flat" {
				var prior ui.Rect
				for _, name := range []string{"Parent session", "Child session", "Last session", "Scratch session"} {
					r, ok := tt.Find(name)
					if !ok || r.H != 32 {
						t.Fatalf("flat row %q %+v", name, r)
					}
					if prior.H > 0 && r.Y-prior.Y != 34 {
						t.Fatalf("flat pitch %v", r.Y-prior.Y)
					}
					prior = r
				}
				return
			}
			p, _ := tt.Find("Parent")
			last, _ := tt.Find("Last")
			scratch, _ := tt.Find("No working directory")
			if p.H != 32 {
				t.Fatalf("header title seat %+v", p)
			}
			if mode == "collapsed" {
				if tt.HasText("Child") || tt.HasText("Parent session") || tt.HasText("Child session") {
					t.Fatal("collapsed descendants remain")
				}
				if last.Y-p.Y != 38 {
					t.Fatalf("closed header pitch %v", last.Y-p.Y)
				}
			} else {
				a, _ := tt.Find("Parent session")
				child, _ := tt.Find("Child")
				b, _ := tt.Find("Child session")
				if a.Y-p.Y != 35 || child.Y-p.Y != 72 || b.Y-a.Y != 72 || last.Y-child.Y != 72 {
					t.Fatalf("group geometry p=%+v a=%+v child=%+v b=%+v last=%+v", p, a, child, b, last)
				}
			}
			c, _ := tt.Find("Last session")
			s, _ := tt.Find("Scratch session")
			if c.Y-last.Y != 35 || scratch.Y-last.Y != 72 || s.Y-scratch.Y != 35 {
				t.Fatalf("scratch pitch last=%+v c=%+v scratch=%+v s=%+v", last, c, scratch, s)
			}
		})
	}
}

func TestSessionDropKeepsEdgeAutoscrollAndEscape(t *testing.T) {
	var items []session.Item
	var ids []string
	for i := 0; i < 35; i++ {
		id := fmt.Sprint(i)
		items = append(items, session.Item{SessionID: id, Title: "Session " + id})
		ids = append(ids, id)
	}
	moves := 0
	m := Model{Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "w", Title: "Workspace", SessionIDs: ids}}, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 1, Collapsed: map[string]bool{"w": false}}}, OnMoveWorkspaceSessionBefore: func(string, string, string) { moves++ }}
	// This case exercises the scroll edge after the user has revealed the roster.
	m.workspaceWatermarks = &workspaceWatermarkState{HostGeneration: 1, Marks: map[string]int{"w": len(items)}}
	var scroll ui.ScrollState
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().TrackScroll(&scroll).Gap(0).Children(func() { workspaceRowsView(c, m, items, ui.RGB(20, 20, 20), ui.RGB(90, 90, 90), ui.Transparent) })
	}, 300, 190)
	s, _ := tt.Find("Session 0")
	tt.Press(s.X+5, s.Y+s.H/2)
	tt.Move(s.X+12, s.Y+s.H/2)
	tt.Move(s.X+5, 185)
	for i := 0; i < 8; i++ {
		tt.Frame()
	}
	if scroll.Y <= 0 {
		t.Fatal("edge autoscroll stopped")
	}
	tt.Key(0, ui.KeyEscape)
	stopped := scroll.Y
	tt.Release(s.X+5, 185)
	tt.Frame()
	if scroll.Y != stopped || moves != 0 {
		t.Fatalf("cancel scroll=%v→%v moves=%d", stopped, scroll.Y, moves)
	}
}
