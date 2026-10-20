package frame

import "github.com/egoist/mygo/ui"

// The original DockSurface floats a tab when a release has no dock target and
// lies outside that surface's root rectangle (DockSurface.tsx:239-243). This is
// an application-content drop, not a global OS pointer hook or a native window.
func workspaceFloatDropSurface(c *ui.Context, m Model, surface ui.Element) {
	if m.Sizes == nil || m.Sizes.Workspace == nil {
		return
	}
	w := m.Sizes.Workspace
	x, y, inside := surface.PointerPosition()
	bounds := surface.Bounds()
	width, height := viewportDIP(c)
	if drag, over := ui.DragOver[workspaceTabDrag](surface); over && inside {
		if n := w.Root.find(drag.ID); n != nil && n.Kind != "project-observer" {
			ui.Overlay(c, func() {
				ui.Box(c).Absolute().Left(min(bounds.X+x+14, max(float32(0), width-180))).Top(min(bounds.Y+y+14, max(float32(0), height-32))).Padding(6).Background(c.Theme().Background).Border(1, c.Theme().Accent).Children(func() { ui.Text(c, paneLabel("放下以浮动面板", "Release to float pane")).FontSize(11) })
			})
		}
	}
	if drag, ok := ui.Drop[workspaceTabDrag](surface); ok && inside {
		px, py := bounds.X+x, bounds.Y+y
		w.pending = func() {
			n := w.Root.find(drag.ID)
			if n == nil || n.Kind == "project-observer" || !w.FloatWorkspaceTab(drag.ID) {
				return
			}
			// Match the original titlebar grab offset while keeping the float visible.
			w.MoveWorkspaceFloat(drag.ID, px-60, py-14)
			visible := boundedFloat(*n.Float, width, height)
			n.Float.X, n.Float.Y = visible.X, visible.Y
		}
	}
}

// Innermost typed targets win. Absorb otherwise-unclaimed drops inside the Dock
// rectangle, so empty pane content cannot bubble into the application fallback.
// Tab insertion and explicit edge targets nested beneath it retain precedence.
func workspaceDockDropBarrier(surface ui.Element) { ui.Drop[workspaceTabDrag](surface) }
