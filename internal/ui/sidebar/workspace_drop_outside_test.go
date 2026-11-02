package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestWorkspaceLastMarkerOutsideReleaseAndCancellation(t *testing.T) {
	for _, mode := range []string{"outside", "escape", "blur", "host", "scope", "removed", "no-marker"} {
		t.Run(mode, func(t *testing.T) {
			groups := []session.WorkspaceGroup{{Key: "a", Kind: "workspace"}, {Key: "b", Kind: "workspace"}, {Key: "c", Kind: "workspace"}}
			state := &sidebarDragState{}
			moves := 0
			m := Model{sidebarDrag: state, Snapshot: session.ViewState{Workspaces: []session.WorkspaceRow{{WorkspaceID: "a"}, {WorkspaceID: "b"}, {WorkspaceID: "c"}}, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 7}}, OnDropWorkspaceBefore: func(g uint64, id, before string) {
				if g != 7 || id != "c" || before != "a" {
					t.Errorf("wrong captured move: %d %s %s", g, id, before)
				}
				moves++
			}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(280).Children(func() { drawDragWorkspaceFixture(c, m, groups, nil) })
			}, 500, 400)
			src, _ := tt.Find("c")
			dst, _ := tt.Find("Header a")
			tt.Press(src.X+10, src.Y+10)
			tt.Move(src.X+22, src.Y+10)
			if mode != "no-marker" {
				tt.Move(dst.X+10, dst.Y+3)
			}
			tt.Move(420, 350)
			switch mode {
			case "escape":
				tt.Key(0, ui.KeyEscape)
			case "blur":
				tt.SetFocused(false)
			case "host":
				m.Snapshot.WorkspaceLayout.HostGeneration++
				tt.Frame()
			case "scope":
				m.Snapshot.WorkspaceLayout.GroupBy = "workspace-tree"
				tt.Frame()
			case "removed":
				m.Snapshot.Workspaces = m.Snapshot.Workspaces[:2]
				tt.Frame()
			}
			tt.Release(420, 350)
			tt.Frame()
			want := 0
			if mode == "outside" {
				want = 1
			}
			if moves != want || state.Workspace != nil {
				t.Fatalf("moves=%d want=%d retained=%+v", moves, want, state.Workspace)
			}
		})
	}
}

func TestSessionLastMarkerOutsideReleaseAndScopeGuards(t *testing.T) {
	for _, mode := range []string{"outside", "escape", "blur", "host", "scope", "cross-account", "pin", "removed"} {
		t.Run(mode, func(t *testing.T) {
			items := []session.Item{{SessionID: "a", Title: "Alpha"}, {SessionID: "b", Title: "Beta"}, {SessionID: "c", Title: "Gamma"}, {SessionID: "d", Title: "Delta"}}
			state := &sidebarDragState{}
			calls := 0
			m := Model{sidebarDrag: state, Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", SessionIDs: []string{"a", "b", "c"}}, {WorkspaceID: "other", SessionIDs: []string{"d"}}}, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 7, Collapsed: map[string]bool{"ws": false, "other": false}}}, OnMoveWorkspaceSessionBefore: func(w, id, before string) {
				if w != "ws" || id != "c" || before != "a" {
					t.Errorf("crossed account or target: %s %s %s", w, id, before)
				}
				calls++
			}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(280).Children(func() { workspaceRowsView(c, m, m.Snapshot.Items, ui.RGB(1, 2, 3), ui.RGB(4, 5, 6), ui.Transparent) })
			}, 500, 500)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			src, ok := tt.Find("Gamma")
			if !ok {
				t.Fatal("source missing")
			}
			dst, ok := tt.Find("Alpha")
			if !ok {
				t.Fatal("target missing")
			}
			tt.Press(src.X+5, src.Y+5)
			tt.Move(src.X+18, src.Y+5)
			tt.Move(dst.X+5, dst.Y+3)
			if mode == "cross-account" {
				other, _ := tt.Find("Delta")
				tt.Move(other.X+5, other.Y+3)
			}
			tt.Move(430, 420)
			switch mode {
			case "escape":
				tt.Key(0, ui.KeyEscape)
			case "blur":
				tt.SetFocused(false)
			case "host":
				m.Snapshot.WorkspaceLayout.HostGeneration++
				tt.Frame()
			case "scope":
				m.Snapshot.WorkspaceLayout.OrderBy = "updated"
				tt.Frame()
			case "pin":
				m.Snapshot.PinnedSessionIDs = []string{"c"}
				m.Snapshot.Items[2].Pinned = true
				tt.Frame()
			case "removed":
				m.Snapshot.Workspaces[0].SessionIDs = []string{"a", "b"}
				tt.Frame()
			}
			tt.Release(430, 420)
			tt.Frame()
			want := 0
			if mode == "outside" {
				want = 1
			}
			if calls != want || state.Session != nil {
				t.Fatalf("calls=%d want=%d retained=%+v", calls, want, state.Session)
			}
		})
	}
}
