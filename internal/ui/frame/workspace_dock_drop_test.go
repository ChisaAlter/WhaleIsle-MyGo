package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestWorkspaceDockDropIdentity(t *testing.T) {
	for _, edge := range []string{"left", "right", "top", "bottom"} {
		t.Run(edge, func(t *testing.T) {
			w := &WorkspacePanes{}
			w.syncRoute("files")
			target := w.Root
			moving := w.selectKind(target, "host-terminal")
			disposed := false
			moving.terminalDispose = func() { disposed = true }
			moving.TerminalSource = "retained"
			if !w.DropWorkspaceTab(moving.ID, target.ID, edge) {
				t.Fatal("drop rejected")
			}
			if w.Root.First == nil || w.Root.find(moving.ID) != moving || moving.TerminalSource != "retained" || disposed {
				t.Fatal("ownership lost")
			}
			if edge == "left" || edge == "top" {
				if w.Root.First != moving {
					t.Fatal("wrong side")
				}
			} else if w.Root.Second != moving {
				t.Fatal("wrong side")
			}
			if !w.DropWorkspaceTab(moving.ID, target.ID, "float") || moving.Float == nil {
				t.Fatal("float rejected")
			}
			if !w.DropWorkspaceTab(moving.ID, target.ID, "tabs") || moving.Float != nil || w.Root.First != nil || disposed {
				t.Fatal("redock failed")
			}
		})
	}
}
func TestWorkspaceDockDropRejectsInvalid(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	before := string(w.Export())
	for _, edge := range []string{"left", "nonsense"} {
		if w.DropWorkspaceTab(w.Root.ID, w.Root.ID, edge) {
			t.Fatal("self drop accepted")
		}
	}
	if string(w.Export()) != before {
		t.Fatal("invalid drop mutated layout")
	}
}

func TestWorkspaceDockDropGestureAndEscape(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	target := w.Root
	moving := w.selectKind(target, "diff")
	s := DefaultLayout()
	s.Workspace = w
	tt := ui.NewTester(func(c *ui.Context) {
		if w.pending != nil {
			f := w.pending
			w.pending = nil
			f()
		}
		ui.Column(c).Fill().Children(func() {
			ui.Button(c, "Drag resource").Drag(workspaceTabDrag{ID: moving.ID})
			workspaceDockDropTargets(c, Model{Sizes: &s, workspacePane: target})
		})
	}, 800, 300)
	drag := func(edge string, cancel bool) {
		src, ok := tt.Find("Drag resource")
		if !ok {
			t.Fatal("missing source")
		}
		dst, ok := tt.Find("Dock drop " + edge)
		if !ok {
			t.Fatal("missing landing target")
		}
		x, y := src.X+src.W/2, src.Y+src.H/2
		tx, ty := dst.X+dst.W/2, dst.Y+dst.H/2
		tt.Press(x, y)
		tt.Move(tx, ty)
		if cancel {
			tt.Key(0, ui.KeyEscape)
		}
		tt.Release(tx, ty)
		// The pending command is applied at the following frame, like workspaceColumns.
		tt.Move(tx+1, ty)
	}
	before := string(w.Export())
	drag("left", true)
	if string(w.Export()) != before {
		t.Fatal("Escape changed workspace")
	}
	drag("left", false)
	if w.Root.First != moving || w.Root.Second != target {
		t.Fatal("drop did not split at target")
	}
	drag("float", false)
	if moving.Float == nil || w.Root.find(moving.ID) != moving {
		t.Fatal("float drop lost resource")
	}
	drag("tabs", false)
	if moving.Float != nil || w.Root.First != nil {
		t.Fatal("float did not redock")
	}
}
