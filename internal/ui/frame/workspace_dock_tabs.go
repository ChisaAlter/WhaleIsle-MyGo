package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// A split-tree leaf is the selected resource. Its siblings are retained here,
// never painted beneath the selected resource's native surface. IDs belong to
// resources, not positions: moving a tab does not create a browser or PTY.
type workspaceTabDrag struct{ ID string }

func (n *WorkspacePane) dockTabs() []*WorkspacePane {
	if n == nil {
		return nil
	}
	index := n.DockIndex
	if index < 0 || index > len(n.DockTabs) {
		index = 0
	}
	tabs := make([]*WorkspacePane, 0, len(n.DockTabs)+1)
	tabs = append(tabs, n.DockTabs[:index]...)
	tabs = append(tabs, n)
	return append(tabs, n.DockTabs[index:]...)
}

func setDockTabs(active *WorkspacePane, tabs []*WorkspacePane) {
	active.DockTabs = nil
	active.DockIndex = 0
	for i, tab := range tabs {
		if tab == active {
			active.DockIndex = i
			continue
		}
		tab.DockTabs = nil
		tab.DockIndex = 0
		active.DockTabs = append(active.DockTabs, tab)
	}
}

func (n *WorkspacePane) dockGroup(id string) *WorkspacePane {
	if n == nil {
		return nil
	}
	if n.First != nil {
		if group := n.First.dockGroup(id); group != nil {
			return group
		}
		return n.Second.dockGroup(id)
	}
	for _, tab := range n.dockTabs() {
		if tab.ID == id {
			return n
		}
	}
	return nil
}

func replaceWorkspaceLeaf(n, old, replacement *WorkspacePane) *WorkspacePane {
	if n == nil || n == old {
		return replacement
	}
	if n.First != nil {
		n.First = replaceWorkspaceLeaf(n.First, old, replacement)
		n.Second = replaceWorkspaceLeaf(n.Second, old, replacement)
	}
	return n
}

func (w *WorkspacePanes) activateDockTab(id string) {
	if n := w.Root.find(id); n != nil && n.Float != nil {
		w.raiseFloat(n)
		return
	}
	group := w.Root.dockGroup(id)
	if group == nil {
		return
	}
	tabs := group.dockTabs()
	for _, tab := range tabs {
		if tab.ID != id {
			continue
		}
		if tab != group {
			group.DockTabs, group.DockIndex = nil, 0
			setDockTabs(tab, tabs)
			w.Root = replaceWorkspaceLeaf(w.Root, group, tab)
			if w.Fullscreen == group.ID {
				w.Fullscreen = tab.ID
			}
		}
		w.Active, w.route = tab.ID, tab.Kind
		return
	}
}

// MoveWorkspaceTab inserts before the destination index; len(tabs) appends.
// Both source and destination are resource IDs, so gestures remain valid when
// a neighboring pane becomes active during the same frame.
func (w *WorkspacePanes) MoveWorkspaceTab(id, destination string, index int) bool {
	for _, resourceID := range []string{id, destination} {
		if n := w.Root.find(resourceID); n != nil && n.Kind == "project-observer" {
			return false
		}
	}
	source, target := w.Root.dockGroup(id), w.Root.dockGroup(destination)
	if source == nil || target == nil {
		return false
	}
	from := source.dockTabs()
	var moving *WorkspacePane
	oldIndex := -1
	for i, tab := range from {
		if tab.ID == id {
			moving, oldIndex = tab, i
			break
		}
	}
	if moving == nil {
		return false
	}
	moving.Float = nil
	if source == target {
		if index > oldIndex {
			index--
		}
		from = append(from[:oldIndex], from[oldIndex+1:]...)
		index = max(0, min(index, len(from)))
		from = append(from, nil)
		copy(from[index+1:], from[index:])
		from[index] = moving
		setDockTabs(source, from)
		w.activateDockTab(id)
		return true
	}
	remaining := append(from[:oldIndex], from[oldIndex+1:]...)
	source.DockTabs, source.DockIndex = nil, 0
	if len(remaining) == 0 {
		w.Root = w.remove(w.Root, source.ID)
	} else {
		selected := source
		if source == moving {
			selected = remaining[min(oldIndex, len(remaining)-1)]
		}
		setDockTabs(selected, remaining)
		w.Root = replaceWorkspaceLeaf(w.Root, source, selected)
	}
	moving.DockTabs, moving.DockIndex = nil, 0
	to := target.dockTabs()
	index = max(0, min(index, len(to)))
	to = append(to, nil)
	copy(to[index+1:], to[index:])
	to[index] = moving
	setDockTabs(target, to)
	w.activateDockTab(id)
	if w.Fullscreen != "" && w.Root.find(w.Fullscreen) == nil {
		w.Fullscreen = w.Active
	}
	return true
}

func (n *WorkspacePane) eachResource(visit func(*WorkspacePane)) {
	n.eachLeaf(func(active *WorkspacePane) {
		for _, tab := range active.dockTabs() {
			visit(tab)
		}
	})
}

func dockTabTitle(n *WorkspacePane) string {
	switch n.Kind {
	case "custom-panels":
		return paneLabel("自定义面板", "Custom Panels")
	case "project-observer":
		return paneLabel("项目概览", "Project overview")
	case "files":
		return paneLabel("文件", "Files")
	case "diff":
		return paneLabel("变更", "Changes")
	case "agents":
		return paneLabel("子代理", "Agents")
	case "browser":
		return paneLabel("浏览器", "Browser")
	case "host-terminal":
		return paneLabel("终端", "Terminal")
	case "plan":
		return paneLabel("计划", "Plan")
	case "schedule-task":
		return paneLabel("任务详情", "Task details")
	case "changes-review":
		return paneLabel("变更审阅", "Changes review")
	case "session-inspector-log":
		return paneLabel("会话检查器", "Session inspector")
	case "subagentchat":
		if n.SubagentChat != nil && n.SubagentChat.Title != "" {
			return n.SubagentChat.Title
		}
		return paneLabel("子代理聊天", "Child chat")
	}
	return n.Kind
}

// DockKit .tabStrip: 10 DIP top inset over a 28 DIP content band. Only the
// tab run shrinks; pane chrome remains reachable at the trailing edge.
func workspaceDockTabStrip(c *ui.Context, m Model) {
	n, w := m.workspacePane, m.Sizes.Workspace
	group := w.Root.dockGroup(n.ID)
	tabs := group.dockTabs()
	n.tabDragging = false
	header := ui.Row(c.Key("workspace-pane-header")).Label("Workspace pane header").FillWidth().MinWidth(0).Height(38).
		Padding(10, 6, 0, 10).Gap(4).AlignItems(ui.Center)
	n.stripWidth = header.Bounds().W
	header.Children(func() {
		tabRun := ui.ScrollHorizontal(c.Key("workspace-pane-tabs")).Grow(1).MinWidth(0).Height(28)
		n.tabRunWidth = tabRun.Bounds().W
		tabRun.Children(func() {
			ui.Row(c).Gap(4).Children(func() {
				for index, tab := range tabs {
					if tab.Float != nil {
						continue
					}
					index, tab := index, tab
					chip := ui.Button(c.Key("resource-tab:"+tab.ID), "").Label(dockTabTitle(tab)).Tooltip(dockTabTitle(tab)).
						Height(28).MinWidth(80).MaxWidth(170).Padding(0, 10).Radius(6).BorderWidth(0).
						Background(ui.Transparent).Drag(workspaceTabDrag{ID: tab.ID}).ContextMenu(workspacePaneMenu(m, tab))
					if tab == n {
						chip.Background(c.Theme().Surface)
					}
					chip.Children(func() {
						ui.Text(c, dockTabTitle(tab)).SingleLine().MinWidth(0).Grow(1).FontSize(theme.InterfaceSize(13))
					})
					if _, over := ui.DragOver[workspaceTabDrag](chip); over {
						chip.Border(2, c.Theme().Accent)
					}
					if dropped, ok := ui.Drop[workspaceTabDrag](chip); ok {
						w.pending = func() { w.MoveWorkspaceTab(dropped.ID, n.ID, index) }
					}
					if chip.Clicked() {
						w.pending = func() { w.activateDockTab(tab.ID) }
					}
					if chip.Dragging() {
						n.tabDragging = true
						chip.Opacity(0.5)
					}
				}
				end := ui.Box(c.Key("resource-tab-end")).Width(10).Height(28)
				if _, over := ui.DragOver[workspaceTabDrag](end); over {
					end.Background(c.Theme().Accent)
				}
				if dropped, ok := ui.Drop[workspaceTabDrag](end); ok {
					w.pending = func() { w.MoveWorkspaceTab(dropped.ID, n.ID, len(tabs)) }
				}
			})
		})
		control := func(key string, icon *ui.SVG, label string) ui.Element {
			button := ui.Button(c.Key(key), "").Label(label).Tooltip(label).Width(28).Height(28).
				Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent)
			button.AlignItems(ui.Center).Justify(ui.Center).Children(func() { ui.Icon(c, icon).Width(16).Height(16).TextColor(c.Theme().TextMuted) })
			return button
		}
		control("workspace-pane-menu", conversationHeaderIcons["EllipsisOutline"], paneLabel("面板操作", "Pane actions")).Menu(workspacePaneMenu(m, n))
		label := paneLabel("面板全屏", "Maximize pane")
		icon := workspacePaneFullscreenIcon
		if w.Fullscreen != "" {
			label = paneLabel("退出全屏", "Restore pane")
			icon = workspacePaneRestoreIcon
		}
		if control("workspace-pane-maximize", icon, label).Clicked() {
			workspacePaneMaximize(m, n)
		}
		if n.Kind != "project-observer" && control("workspace-pane-close", workspacePaneCloseIcon, paneLabel("关闭标签", "Close tab")).Clicked() {
			workspacePaneClose(m, n)
		}
	})
}

// Opening adapters sometimes attach a newly resolved resource to the source
// seat before requesting its route. Transfer only that resource's ownership;
// an existing Files editor or terminal occurrence must never be cloned here.
func transferDockResource(target, source *WorkspacePane, kind string) {
	if source == target || source.Kind == kind {
		return
	}
	switch kind {
	case "browser":
		if source.browser != nil {
			target.browser, source.browser = source.browser, nil
		}
		if source.Browser != nil {
			target.Browser, source.Browser = source.Browser, nil
		}
	case "diff":
		if source.Diff != nil {
			target.Diff, source.Diff = source.Diff, nil
		}
	case "plan":
		if source.Files != nil {
			target.Files = &WorkspaceFileState{Tabs: source.Files.Tabs.CopyPlanPane()}
		}
	case "changes-review":
		if source.ChangesReview != nil {
			target.ChangesReview, source.ChangesReview = source.ChangesReview, nil
		}
	case "session-inspector-log":
		if source.Inspector != nil {
			target.Inspector, source.Inspector = source.Inspector, nil
		}
	case "schedule-task":
		if source.Schedule != nil {
			target.Schedule, source.Schedule = source.Schedule, nil
		}
	case "subagentchat":
		if source.SubagentChat != nil {
			target.SubagentChat, source.SubagentChat = source.SubagentChat, nil
		}
	}
}
