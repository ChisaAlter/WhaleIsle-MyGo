package frame

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Dragging the dock split divider must update the persisted ratio. The bound
// stack local used to be read back before applyInputs flushed the drag, so
// the divider snapped back.
func TestWorkspaceDockSplitDragUpdatesRatio(t *testing.T) {
	sizes := DefaultLayout()
	w := &WorkspacePanes{}
	sizes.Workspace = w
	n := &WorkspacePane{
		ID:     "dock",
		Axis:   "horizontal",
		Ratio:  .5,
		First:  w.leaf("guide"),
		Second: w.leaf("guide"),
	}
	m := Model{Sizes: &sizes}
	tt := ui.NewTester(func(c *ui.Context) {
		workspaceDockNode(c, m, n)
	}, 800, 400)
	tt.Press(400, 200)
	tt.Move(600, 200)
	tt.Release(600, 200)
	if math.Abs(float64(n.Ratio-.75)) > .05 {
		t.Fatalf("drag did not update ratio: %v", n.Ratio)
	}
}

// The vertical dock split has the same defect through SplitVertical.
func TestWorkspaceDockVerticalSplitDragUpdatesRatio(t *testing.T) {
	sizes := DefaultLayout()
	w := &WorkspacePanes{}
	sizes.Workspace = w
	n := &WorkspacePane{
		ID:     "dock-v",
		Axis:   "vertical",
		Ratio:  .5,
		First:  w.leaf("guide"),
		Second: w.leaf("guide"),
	}
	m := Model{Sizes: &sizes}
	tt := ui.NewTester(func(c *ui.Context) {
		workspaceDockNode(c, m, n)
	}, 800, 400)
	tt.Press(400, 200)
	tt.Move(400, 300)
	tt.Release(400, 300)
	if math.Abs(float64(n.Ratio-.75)) > .05 {
		t.Fatalf("drag did not update ratio: %v", n.Ratio)
	}
}
