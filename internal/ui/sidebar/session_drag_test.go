package sidebar

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestSessionDragAnchor(t *testing.T) {
	rows := []session.WorkspaceSessionRow{{Item: session.Item{SessionID: "a"}}, {Item: session.Item{SessionID: "b"}}, {Item: session.Item{SessionID: "c"}}}
	order := []string{"a", "hidden", "b", "c"}
	if got, ok := sessionDragAnchor(order, rows, "c", "a", true); !ok || got != "hidden" {
		t.Fatalf("after anchor must preserve hidden members: %q %v", got, ok)
	}
	if got, ok := sessionDragAnchor(order, rows, "a", "c", true); !ok || got != "" {
		t.Fatalf("append: %q %v", got, ok)
	}
	if _, ok := sessionDragAnchor(order, rows, "a", "b", false); ok {
		t.Fatal("visually unchanged move accepted")
	}
	rows[2].Item.Pinned = true
	if _, ok := sessionDragAnchor(order, rows, "c", "a", false); ok {
		t.Fatal("cross pinned partition accepted")
	}
	rows[2].Item.Pinned = false
	rows[2].Item.Blank = true
	if _, ok := sessionDragAnchor(order, rows, "c", "a", false); ok {
		t.Fatal("blank drag accepted")
	}
	rows[2].Item.Blank = false
	rows[2].Depth = 1
	if _, ok := sessionDragAnchor(order, rows, "c", "a", false); ok {
		t.Fatal("nested drag accepted")
	}
	if _, ok := sessionDragAnchor(order, rows, "missing", "a", false); ok {
		t.Fatal("unknown accepted")
	}
}

func TestSessionManualPins(t *testing.T) {
	got := sessionManualPins([]string{"b", "hidden", "a"}, []string{"a", "b", "other"})
	if len(got) != 3 || got[0] != "b" || got[1] != "a" || got[2] != "other" {
		t.Fatalf("pin partition order: %v", got)
	}
}

func TestSessionDragUI(t *testing.T) {
	items := []session.Item{{SessionID: "a", Title: "Alpha"}, {SessionID: "b", Title: "Beta"}, {SessionID: "c", Title: "Gamma"}}
	var calls int
	var moved, anchor, owner string
	m := Model{Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", SessionIDs: []string{"a", "b", "c"}}}}, OnMoveWorkspaceSessionBefore: func(w, id, before string) { calls++; owner = w; moved = id; anchor = before }}
	m.Snapshot.WorkspaceLayout.HostGeneration = 7
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(1, 2, 3), ui.RGB(4, 5, 6), ui.Transparent) })
	}, 300, 400)
	src, ok := tt.Find("↳ Gamma")
	if !ok {
		t.Fatal("source missing")
	}
	dst, ok := tt.Find("↳ Alpha")
	if !ok {
		t.Fatal("target missing")
	}
	tt.Press(src.X+5, src.Y+5)
	tt.Move(src.X+15, src.Y+5)
	tt.Move(dst.X+5, dst.Y+5)
	tt.Release(dst.X+5, dst.Y+5)
	if calls != 1 || owner != "ws" || moved != "c" || anchor != "a" {
		t.Fatalf("drop did not reach persisted callback once: %d %s %s %s", calls, owner, moved, anchor)
	}
	tt.Press(src.X+5, src.Y+5)
	tt.Move(src.X+15, src.Y+5)
	tt.Move(dst.X+5, dst.Y+5)
	tt.Key(0, ui.KeyEscape)
	tt.Release(dst.X+5, dst.Y+5)
	if calls != 1 {
		t.Fatal("Escape committed a reorder")
	}
}
