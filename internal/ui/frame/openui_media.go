package frame

import "github.com/egoist/mygo"

// Chat media sits below floating resources and native browser mini-players.
// These are the same bounded workspace rectangles used by browser occlusion.
// NativeSurfaceClip separately excludes scroll ancestors and active modals.
func openUIMediaOcclusions(m Model) []mygo.Rectangle {
	if m.OpenUIWindow == nil || m.OpenUIWindow.IsDestroyed() || m.Sizes == nil || m.Sizes.Workspace == nil || m.Sizes.Workspace.Root == nil {
		return nil
	}
	bounds := m.OpenUIWindow.ContentBounds()
	var holes []mygo.Rectangle
	m.Sizes.Workspace.Root.eachResource(func(n *WorkspacePane) {
		if n.Float != nil && (n.Kind != "project-observer" || projectObserverVisible(m, n)) {
			f := boundedFloat(*n.Float, float32(bounds.Width), float32(bounds.Height))
			holes = append(holes, mygo.Rectangle{X: int(f.X), Y: int(f.Y), Width: int(f.Width), Height: int(f.Height)})
		}
		if n.browser != nil {
			if r, ok := n.browser.MiniPlayerBounds(); ok {
				holes = append(holes, r)
			}
		}
	})
	return holes
}
