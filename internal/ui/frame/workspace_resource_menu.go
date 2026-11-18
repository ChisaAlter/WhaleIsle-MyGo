package frame

import (
	"sort"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	termui "whaleisle/internal/ui/terminal"
)

type workspaceResourceEntry struct {
	id, title, description string
	icon                   *ui.SVG
	disabled               bool
	open                   func()
}

func workspaceResourceIcon(name string) *ui.SVG {
	switch name {
	case "terminal", "PluginArtworkTerminal":
		return workspacePaneTerminalIcon
	case "browser", "GuideArtworkBrowser":
		return workspacePaneBrowserIcon
	case "code", "CodeOutline":
		return conversationHeaderIcons["CodeOutline"]
	case "agents", "AgentPresetOutline", "IconAgentPresetOutline16":
		return conversationHeaderIcons["AgentPresetOutline"]
	case "team", "UsersOutline":
		return conversationHeaderIcons["UsersOutline"]
	case "files", "folder", "PanelLeftOutline":
		return conversationHeaderIcons["PanelLeftOutline"]
	default:
		return workspacePaneResourcesIcon
	}
}

// One inventory backs the guide and its explicit resource-menu shortcut.
// Pane operations never double as the only door into resource content.
func workspaceResourceEntries(m Model) []workspaceResourceEntry {
	var entries []workspaceResourceEntry
	order := map[string]int{"files": 10, "host-terminal": 20, "browser": 30, "diff": 30, "agents": 40, "team": 45, "session-inspector-log": 60}
	route := func(kind, title, description string, icon *ui.SVG, disabled bool) {
		entries = append(entries, workspaceResourceEntry{kind, title, description, icon, disabled || m.RightPaneTab == nil, func() { openWorkspaceResource(m, kind, nil) }})
	}
	selected := m.Store != nil && m.Store.SelectedSessionID() != ""
	standalone := m.Sizes == nil || m.Sizes.Workspace == nil
	route("files", i18n.T(i18n.KeyRightPaneFiles), surfacesText("card.files.description"), conversationHeaderIcons["PanelLeftOutline"], false)
	route("diff", i18n.T(i18n.KeyRightPaneDiff), surfacesText("card.diff.description"), conversationHeaderIcons["CodeOutline"], false)
	route("agents", i18n.T(i18n.KeyRightPaneAgents), surfacesText("card.agents.description"), conversationHeaderIcons["AgentPresetOutline"], !selected && !standalone)
	if standalone {
		route("team", paneLabel("团队", "Team"), "", conversationHeaderIcons["UsersOutline"], false)
	} else {
		entries = append(entries, ordinaryTeamResourceEntry(m))
	}
	if m.CustomPanelContent != nil {
		route("custom-panels", paneLabel("面板", "Panels"), paneLabel("打开自定义面板", "Open custom panels"), workspacePaneResourcesIcon, false)
	}
	if !m.DisableDeveloperTools {
		route("session-inspector-log", paneLabel("会话日志", "Session Log"), paneLabel("检查当前会话原始事件", "Inspect this session's raw events"), conversationHeaderIcons["CodeOutline"], !selected)
	}
	if m.workspacePane != nil && m.workspacePane.ChangesReview != nil {
		route("changes-review", changesReviewTitle(m), "", workspacePaneResourcesIcon, false)
	}
	if m.workspacePane != nil && m.workspacePane.SubagentChat != nil {
		route("subagentchat", subagentChatPaneTitle(m), "", conversationHeaderIcons["AgentPresetOutline"], false)
	}
	if m.FileTabs != nil && len(m.FileTabs.Plans.Tabs) > 0 {
		route("plan", paneLabel("计划", "Plan"), "", workspacePaneResourcesIcon, false)
	}
	if m.workspacePane != nil && m.workspacePane.Schedule != nil {
		route("schedule-task", schedulePaneTitle(m), "", conversationHeaderIcons["ClockOutline"], false)
	}
	if m.HostTerminalStore != nil {
		entries = append(entries, workspaceResourceEntry{"host-terminal", termui.HostTitle(), surfacesText("card.terminal.description"), workspacePaneTerminalIcon, !selected, func() {
			openWorkspaceResource(m, "host-terminal", m.OnNewHostTerminal)
		}})
	}
	if standalone && m.OnOpenBrowserPreview != nil {
		entries = append(entries, workspaceResourceEntry{"browser", i18n.T(i18n.KeyBrowserPreviewTitle), "", workspacePaneBrowserIcon, false, m.OnOpenBrowserPreview})
	} else if m.BrowserContent != nil || m.NewWorkspaceBrowser != nil {
		route("browser", i18n.T(i18n.KeyBrowserPreviewTitle), surfacesText("card.browser.description"), workspacePaneBrowserIcon, false)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		left, ok := order[entries[i].id]
		if !ok {
			left = 100
		}
		right, ok := order[entries[j].id]
		if !ok {
			right = 100
		}
		return left < right
	})
	return entries
}

// The explicit resource menu is separate from both Session More and tab actions.
// DockKit's add control opens the guide, not this native menu.
func workspaceResourceMenu(menu *ui.Menu, m Model) {
	for _, entry := range workspaceResourceEntries(m) {
		if menu.Item(entry.title).Disabled(entry.disabled).Chosen() {
			entry.open()
		}
	}
	if m.Sizes == nil || m.Sizes.Workspace == nil {
		if m.OnOpenMarket != nil && menu.Item(paneLabel("浏览更多插件", "Browse other plugins")).Chosen() {
			m.OnOpenMarket()
		}
		if m.OnCreatePluginDraft != nil && menu.Item(paneLabel("创建你的插件", "Create your plugin")).Chosen() {
			m.OnCreatePluginDraft("rightbar")
		}
	}
}
func openWorkspaceResource(m Model, kind string, open func()) {
	if m.Sizes == nil || m.Sizes.Workspace == nil {
		if open != nil {
			open()
		} else if m.RightPaneTab != nil {
			*m.RightPaneTab = kind
		}
		return
	}
	w := m.Sizes.Workspace
	sourceID := w.Active
	if m.workspacePane != nil {
		sourceID = m.workspacePane.ID
	}
	w.pending = func() {
		source := w.Root.find(sourceID)
		if source == nil {
			return
		}
		w.activateDockTab(source.ID)
		if open != nil {
			open()
			if source.Kind == "guide" && w.Active != source.ID {
				w.disposeWorkspaceResource(source.ID)
			}
		} else {
			w.selectKind(source, kind)
		}
		if m.RightPaneTab != nil {
			*m.RightPaneTab = w.route
		}
	}
}

func (w *WorkspacePanes) openGuide(sourceID string) {
	group := w.Root.dockGroup(sourceID)
	if group == nil {
		return
	}
	for _, tab := range group.dockTabs() {
		if tab.Kind == "guide" && tab.Float == nil {
			w.activateDockTab(tab.ID)
			return
		}
	}
	guide := w.leaf("guide")
	setDockTabs(group, append(group.dockTabs(), guide))
	w.activateDockTab(guide.ID)
}
