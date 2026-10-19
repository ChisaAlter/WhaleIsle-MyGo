package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// DropWorkspaceTab relocates an existing occurrence. It never closes, clones,
// or reopens its resource. Invalid/canceled gestures leave the tree untouched.
func (w *WorkspacePanes) DropWorkspaceTab(id, destination, edge string) bool {
	if w == nil || w.Root == nil {
		return false
	}
	moving, target := w.Root.find(id), w.Root.find(destination)
	if moving == nil || moving.First != nil || moving.Kind == "project-observer" || target == nil || target.Kind == "project-observer" {
		return false
	}
	if edge == "float" {
		return w.FloatWorkspaceTab(id)
	}
	if id == destination || target.Float != nil {
		return false
	}
	if edge == "tabs" {
		return w.MoveWorkspaceTab(id, destination, len(w.Root.dockGroup(destination).dockTabs()))
	}
	axis := "horizontal"
	switch edge {
	case "left", "right":
	default:
		return false
	}
	source := w.Root.dockGroup(id)
	if source == nil {
		return false
	}
	if !w.canSplit(w.Root.dockGroup(destination)) {
		return false
	}
	tabs := source.dockTabs()
	remaining := make([]*WorkspacePane, 0, len(tabs)-1)
	for _, tab := range tabs {
		if tab != moving {
			remaining = append(remaining, tab)
		}
	}
	source.DockTabs, source.DockIndex = nil, 0
	if len(remaining) == 0 {
		w.Root = w.remove(w.Root, source.ID)
	} else {
		selected := source
		if selected == moving {
			selected = remaining[0]
		}
		setDockTabs(selected, remaining)
		w.Root = replaceWorkspaceLeaf(w.Root, source, selected)
	}
	moving.DockTabs, moving.DockIndex, moving.Float = nil, 0, nil
	target = w.Root.dockGroup(destination)
	split := w.leaf("")
	split.Axis, split.Ratio = axis, .5
	split.First, split.Second = target, moving
	if edge == "left" || edge == "top" {
		split.First, split.Second = moving, target
	}
	w.Root = replaceWorkspaceLeaf(w.Root, target, split)
	w.Active, w.route, w.Fullscreen = moving.ID, moving.Kind, ""
	return true
}

// PaneDropHints lives above the body only during a tab gesture. The pane's
// own typed destination discovers an incoming drag; the saved flag keeps its
// hints mounted while the pointer is over one of those child destinations.
func workspaceDockDropTargets(c *ui.Context, m Model, surface ui.Element) {
	w, n := m.Sizes.Workspace, m.workspacePane
	_, overPane := ui.DragOver[workspaceTabDrag](surface)
	visible := ui.Local(surface, "workspace-dock-hints", func() bool { return false })
	show := overPane || n.tabDragging || *visible
	if !show {
		return
	}
	childOver := false
	width, height := surface.Bounds().W, max(float32(0), surface.Bounds().H-38)
	ui.Box(c.Key("workspace-dock-hints")).Absolute().Left(0).Right(0).Top(38).Bottom(0).PassThrough().Children(func() {
		for _, item := range []struct {
			edge, zh, en string
			x, y, w, h   float32
		}{
			{"left", "左侧", "Left", 0, 0, width * .4, height},
			{"right", "右侧", "Right", width * .6, 0, width * .4, height},
			{"tabs", "标签组", "Tabs", width * .4, 0, width * .2, height},
		} {
			item := item
			if item.edge != "tabs" && !w.canSplit(n) {
				continue
			}
			zone := ui.Box(c.Key("dock-drop:" + item.edge)).Absolute().Left(item.x).Top(item.y).
				Width(item.w).Height(item.h).Padding(8).Label("Dock drop " + item.edge)
			_, over := ui.DragOver[workspaceTabDrag](zone)
			if over {
				childOver = true
			}
			// Original dockHint reserves a 40% side or the whole center;
			// feedback paints only the active destination over the body.
			if over {
				zone.Children(func() {
					ui.Box(c).Fill().Radius(12).Background(c.Theme().Surface).Border(1.5, c.Theme().Accent).
						AlignItems(ui.Center).Justify(ui.Center).Children(func() { ui.Text(c, paneLabel(item.zh, item.en)).FontSize(theme.InterfaceSize(13)).FontWeight(500) })
				})
			}
			if payload, ok := ui.Drop[workspaceTabDrag](zone); ok {
				w.pending = func() { w.DropWorkspaceTab(payload.ID, n.ID, item.edge) }
			}
		}
	})
	keep := overPane || n.tabDragging || childOver
	if *visible != keep {
		*visible = keep
		c.Invalidate()
	}
}
