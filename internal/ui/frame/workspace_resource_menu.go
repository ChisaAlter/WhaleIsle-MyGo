package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	termui "whaleisle/internal/ui/terminal"
)

// Resource kinds occupy DockKit's menu, not a second permanent tab strip.
// Keep the existing route callbacks and occurrence-specific resource guards.
func workspaceResourceMenu(menu *ui.Menu, m Model) {
	route := func(kind, label string) {
		if menu.Item(label).Disabled(m.RightPaneTab == nil).Chosen() && m.RightPaneTab != nil {
			*m.RightPaneTab = kind
		}
	}
	route("files", i18n.T(i18n.KeyRightPaneFiles))
	route("diff", i18n.T(i18n.KeyRightPaneDiff))
	route("agents", i18n.T(i18n.KeyRightPaneAgents))
	if m.CustomPanelContent != nil {
		route("custom-panels", paneLabel("面板", "Panels"))
	}
	if !m.DisableDeveloperTools {
		route("session-inspector-log", paneLabel("会话日志 · 检查当前会话原始事件", "Session Log · Inspect this session's raw events"))
	}
	if m.workspacePane != nil && m.workspacePane.ChangesReview != nil {
		route("changes-review", changesReviewTitle(m))
	}
	if m.workspacePane != nil && m.workspacePane.SubagentChat != nil {
		route("subagentchat", subagentChatPaneTitle(m))
	}
	if m.FileTabs != nil && len(m.FileTabs.Plans.Tabs) > 0 {
		route("plan", paneLabel("计划", "Plan"))
	}
	if m.workspacePane != nil && m.workspacePane.Schedule != nil {
		route("schedule-task", schedulePaneTitle(m))
	}
	if m.HostTerminalStore != nil {
		route("host-terminal", termui.HostTitle())
	}
	if m.OnOpenBrowserPreview != nil && menu.Item(i18n.T(i18n.KeyBrowserPreviewTitle)).Chosen() {
		m.OnOpenBrowserPreview()
	}
	if m.OnOpenMarket != nil && menu.Item(paneLabel("浏览更多插件", "Browse other plugins")).Chosen() {
		m.OnOpenMarket()
	}
	if m.OnCreatePluginDraft != nil && menu.Item(paneLabel("创建你的插件", "Create your plugin")).Chosen() {
		m.OnCreatePluginDraft("rightbar")
	}
}
