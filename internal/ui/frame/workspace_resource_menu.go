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

// PluginRightPaneKind describes one registered guide door, not an open tab.
// The source keeps responsibility for creating/mounting its real occurrence.
type PluginRightPaneKind struct {
	ID, ProviderID, Kind, Label, Description, Icon string
	Order                                          int
	Disabled                                       bool
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
		entries = append(entries, workspaceResourceEntry{"host-terminal", termui.HostTitle(), surfacesText("card.terminal.description"), conversationHeaderIcons["CodeOutline"], !selected, func() {
			openWorkspaceResource(m, "host-terminal", m.OnNewHostTerminal)
		}})
	}
	if standalone && m.OnOpenBrowserPreview != nil {
		entries = append(entries, workspaceResourceEntry{"browser", i18n.T(i18n.KeyBrowserPreviewTitle), "", workspacePaneBrowserIcon, false, m.OnOpenBrowserPreview})
	} else if m.BrowserContent != nil || m.NewWorkspaceBrowser != nil {
		route("browser", i18n.T(i18n.KeyBrowserPreviewTitle), surfacesText("card.browser.description"), workspacePaneResourcesIcon, false)
	}
	for _, registered := range m.PluginRightPaneKinds {
		if registered.ID == "" || registered.Kind == "" || registered.Label == "" || m.DisableDeveloperTools && registered.Kind == "session-inspector-log" {
			continue
		}
		kind := registered.Kind
		if kind == "terminal" {
			kind = "host-terminal"
		}
		if kind == "agent-team" {
			kind = "team"
		}
		nativeProviders := map[string]string{
			"@deepseek-ai/dsh-client-ui-files/files":             "files",
			"@deepseek-ai/dsh-client-ui-sidebar-files":           "files",
			"@deepseek-ai/dsh-client-ui-diff":                    "diff",
			"@deepseek-ai/dsh-client-ui-agents-panel":            "agents",
			"@deepseek-ai/dsh-experimental-client-ui-agent-team": "team",
			"@deepseek-ai/dsh-client-ui-sidebar-terminal":        "host-terminal",
			"@deepseek-ai/dsh-client-ui-sidebar-browser":         "browser",
			"@deepseek-ai/dsh-client-ui-preview/browser":         "browser",
			"@deepseek-ai/dsh-experimental-session-inspector":    "session-inspector-log",
		}
		native := false
		for i := range entries {
			if entries[i].id == kind {
				if nativeProviders[registered.ProviderID] != kind {
					entries = append(entries[:i], entries[i+1:]...)
					break
				}
				entries[i].title, entries[i].description = registered.Label, registered.Description
				entries[i].disabled = entries[i].disabled || registered.Disabled
				if registered.Icon != "" {
					entries[i].icon = workspaceResourceIcon(registered.Icon)
				}
				order[kind] = registered.Order
				native = true
				break
			}
		}
		if native {
			continue
		}
		registered := registered
		id := "registered:" + registered.ID
		order[id] = registered.Order
		entries = append(entries, workspaceResourceEntry{id, registered.Label, registered.Description, workspaceResourceIcon(registered.Icon), registered.Disabled || !selected || m.OnOpenPluginRightPaneKind == nil, func() {
			guideID := ""
			if m.workspacePane != nil && m.workspacePane.Kind == "guide" {
				guideID = m.workspacePane.ID
			}
			if err := m.OnOpenPluginRightPaneKind(registered.Kind, guideID); err != nil && m.Sizes != nil && m.Sizes.Workspace != nil {
				m.Sizes.Workspace.message = err.Error()
			}
		}})
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
	if len(m.PluginRightPaneTabs) > 0 {
		menu.Separator()
	}
	for _, plugin := range m.PluginRightPaneTabs {
		if plugin.Handle == "" {
			continue
		}
		plugin := plugin
		label := plugin.Label
		if label == "" {
			label = plugin.Kind
		}
		if menu.Item(label).Disabled(m.OnSelectPluginRightPaneTab == nil || m.Sizes == nil || m.Sizes.Workspace == nil).Chosen() {
			openWorkspaceResource(m, "plugin-native", func() {
				m.OnSelectPluginRightPaneTab(plugin.Handle)
				m.Sizes.Workspace.OpenPluginRightPaneTab(plugin.Handle, label)
			})
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
