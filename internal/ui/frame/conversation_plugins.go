package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// selectedSessionPluginAction rejects a menu left open across selection changes.
// Both header and row access operate only on the exact selected Session owner.
func selectedSessionPluginAction(m Model, selected string, action func()) func(string) {
	if action == nil || selected == "" || m.Store == nil {
		return nil
	}
	return func(id string) {
		if id == selected && m.Store.SelectedSessionID() == selected {
			action()
		}
	}
}

// The installed session-log-export action owns this menu. View selection,
// plugin discovery and the launcher have separate entry points.
func conversationMoreMenu(c *ui.Context, m Model, snapshot session.ViewState, ink ui.Color) {
	if m.ConversationContributions == nil && (m.OnOpenSessionPlugins != nil || m.OnOpenLauncher != nil || m.HideChatViewTabs && m.ChatPaneTab != nil || m.OnOpenMarket != nil || m.OnCreatePluginDraft != nil) {
		legacyConversationMoreMenu(c, m, ink)
		return
	}
	id := snapshot.SelectedID
	if id == "" || conversationIsManaged(snapshot) {
		return
	}
	button := conversationHeaderButton(c, "conversation-more", paneLabel("更多操作", "More actions"), "EllipsisOutline", true, ink, func() {}, 0)
	button.Menu(func(menu *ui.Menu) {
		current := m.Store != nil && m.Store.SelectedSessionID() == id
		busy := m.SessionLogDownloading != nil && m.SessionLogDownloading(id)
		if menu.Item(paneLabel("下载会话日志", "Download session log")).Disabled(!current || busy || m.OnDownloadSessionLog == nil).Chosen() {
			m.OnDownloadSessionLog(id)
		}
		if m.OnOpenSessionFeedback != nil && menu.Item(paneLabel("反馈", "Feedback")).Disabled(!current).Chosen() {
			m.OnOpenSessionFeedback(id)
		}
	})
}

// Standalone embeddings without the source header publisher retain their
// existing explicit view/plugin/launcher callbacks. The production publisher
// owns those doors separately from its session-log menu.
func legacyConversationMoreMenu(c *ui.Context, m Model, ink ui.Color) {
	hasPlugins := m.OnOpenSessionPlugins != nil
	hasViews := m.HideChatViewTabs && m.ChatPaneTab != nil
	if !hasPlugins && !hasViews && m.OnOpenMarket == nil && m.OnCreatePluginDraft == nil {
		conversationHeaderButton(c, "conversation-more", i18n.T(i18n.KeyLauncherOpen), "EllipsisOutline", true, ink, m.OnOpenLauncher, 0)
		return
	}
	button := conversationHeaderButton(c, "conversation-more", i18n.T(i18n.KeyLauncherOpen), "EllipsisOutline", true, ink, func() {}, 0)
	button.Menu(func(menu *ui.Menu) {
		if hasViews {
			current := *m.ChatPaneTab
			if current == "" {
				current = "chat"
			}
			for _, view := range []struct{ id, label string }{{"chat", paneLabel("对话", "Chat")}, {"trajectory", paneLabel("轨迹", "Trajectory")}} {
				label := view.label
				if current == view.id {
					label = "✓ " + label
				}
				if menu.Item(label).Chosen() {
					*m.ChatPaneTab = view.id
					if m.OnSelectBuiltinConversationView != nil {
						m.OnSelectBuiltinConversationView(view.id)
					}
				}
			}
			for _, view := range m.ConversationViews {
				if menu.Item(view.Label).Disabled(m.OnSelectConversationView == nil).Chosen() {
					*m.ChatPaneTab = "plugin:" + view.Handle
					m.OnSelectConversationView(view.Handle)
				}
			}
		}
		if hasPlugins && menu.Item(i18n.T(i18n.KeyCmdSessionPluginTitle)).Chosen() {
			m.OnOpenSessionPlugins()
		}
		if hasPlugins && m.SessionPluginsActive && m.OnCloseSessionPlugins != nil && menu.Item(i18n.T(i18n.KeySessionPluginsStop)).Chosen() {
			m.OnCloseSessionPlugins()
		}
		if m.OnOpenMarket != nil && menu.Item(paneLabel("浏览更多插件", "Browse other plugins")).Chosen() {
			m.OnOpenMarket()
		}
		if m.OnCreatePluginDraft != nil && menu.Item(paneLabel("创建你的插件", "Create your plugin")).Chosen() {
			m.OnCreatePluginDraft("conversation")
		}
		if m.OnOpenLauncher != nil && menu.Item(i18n.T(i18n.KeyLauncherOpen)).Chosen() {
			m.OnOpenLauncher()
		}
	})
}
