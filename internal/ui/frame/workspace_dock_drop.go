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
	case "top", "bottom":
		axis = "vertical"
	default:
		return false
	}
	source := w.Root.dockGroup(id)
	if source == nil {
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

// The labeled landing strip remains stable during the gesture, including in
// narrow docks. No mutation occurs on hover; ui's value drag cancels on Escape.
func workspaceDockDropTargets(c *ui.Context, m Model) {
	w, n := m.Sizes.Workspace, m.workspacePane
	ui.ScrollHorizontal(c).FillWidth().Height(29).Children(func() {
		ui.Row(c).Gap(3).Padding(2).Children(func() {
			for _, item := range []struct{ edge, zh, en string }{
				{"left", "← 左侧", "← Left"}, {"right", "右侧 →", "Right →"},
				{"top", "↑ 上方", "↑ Above"}, {"bottom", "↓ 下方", "↓ Below"},
				{"tabs", "标签组", "Tabs"}, {"float", "↗ 浮窗", "↗ Float"},
			} {
				item := item
				zone := ui.Box(c).Key("dock-drop:"+item.edge).Height(24).Padding(3).Border(1, c.Theme().Border).Label("Dock drop " + item.edge)
				zone.Children(func() { ui.Text(c, paneLabel(item.zh, item.en)).FontSize(theme.InterfaceSize(10)) })
				if _, over := ui.DragOver[workspaceTabDrag](zone); over {
					zone.Border(2, c.Theme().Accent).Background(c.Theme().Background)
				}
				if payload, ok := ui.Drop[workspaceTabDrag](zone); ok {
					w.pending = func() { w.DropWorkspaceTab(payload.ID, n.ID, item.edge) }
				}
			}
		})
	})
}
