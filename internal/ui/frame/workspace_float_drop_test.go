package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestWorkspaceFreeFloatDrop(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	moving := w.Root
	target := w.selectKind(moving, "diff")
	sizes := DefaultLayout()
	sizes.Workspace = w
	tt := ui.NewTester(func(c *ui.Context) {
		shell := ui.Column(c).Fill()
		workspaceFloatDropSurface(c, Model{Sizes: &sizes}, shell)
		shell.Children(func() {
			dock := ui.Column(c).Width(300).Height(200)
			workspaceDockDropBarrier(dock)
			dock.Children(func() {
				ui.Button(c, "Move resource").Drag(workspaceTabDrag{ID: moving.ID})
				workspaceDockDropTargets(c, Model{Sizes: &sizes, workspacePane: target})
			})
		})
		if w.pending != nil {
			f := w.pending
			w.pending = nil
			f()
		}
	}, 800, 500)
	drag := func(x, y float32, cancel bool) {
		src, _ := tt.Find("Move resource")
		tt.Press(src.X+src.W/2, src.Y+src.H/2)
		tt.Move(x, y)
		if cancel {
			tt.Key(0, ui.KeyEscape)
		}
		tt.Release(x, y)
		tt.Move(x+1, y)
	}
	drag(150, 170, false)
	if moving.Float != nil {
		t.Fatal("Dock interior blank incorrectly floated")
	}
	drag(650, 350, true)
	if moving.Float != nil {
		t.Fatal("Escape committed free float")
	}
	drag(650, 350, false)
	if moving.Float == nil || w.Root.find(moving.ID) != moving {
		t.Fatal("application blank did not float original occurrence")
	}
	if moving.Float.X < 400 || moving.Float.Y < 200 {
		t.Fatalf("did not place near release: %+v", moving.Float)
	}
	// A specific child destination still wins over the fallback surface.
	zone, _ := tt.Find("Dock drop tabs")
	drag(zone.X+zone.W/2, zone.Y+zone.H/2, false)
	if moving.Float != nil {
		t.Fatal("explicit target lost precedence")
	}
}
