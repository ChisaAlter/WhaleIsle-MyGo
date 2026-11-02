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

// conversationMoreMenu keeps optional plugin access in normal header flow.
// Its visibility does not depend on creating a browser to inspect a Session.
func conversationMoreMenu(c *ui.Context, m Model, snapshot session.ViewState, ink ui.Color) {
	hasSessionPlugins := m.OnOpenSessionPlugins != nil && snapshot.SelectedID != ""
	hasFoldedViews := m.HideChatViewTabs && m.ChatPaneTab != nil
	hasEntryActions := m.OnOpenMarket != nil || m.OnCreatePluginDraft != nil
	if !hasSessionPlugins && !hasFoldedViews && !hasEntryActions {
		conversationHeaderButton(c, "conversation-more", i18n.T(i18n.KeyLauncherOpen), "EllipsisOutline", true, ink, m.OnOpenLauncher, 0)
		return
	}
	button := conversationHeaderButton(c, "conversation-more", i18n.T(i18n.KeyLauncherOpen), "EllipsisOutline", true, ink, func() {}, 0)
	button.Menu(func(menu *ui.Menu) {
		if hasFoldedViews {
			cur := *m.ChatPaneTab
			if cur == "" {
				cur = "chat"
			}
			chatLabel := paneLabel("对话", "Chat")
			if cur == "chat" {
				chatLabel = "✓ " + chatLabel
			}
			if menu.Item(chatLabel).Chosen() {
				*m.ChatPaneTab = "chat"
			}
			trajLabel := paneLabel("轨迹", "Trajectory")
			if cur == "trajectory" {
				trajLabel = "✓ " + trajLabel
			}
			if menu.Item(trajLabel).Chosen() {
				*m.ChatPaneTab = "trajectory"
			}
		}
		if hasSessionPlugins {
			if menu.Item(i18n.T(i18n.KeyCmdSessionPluginTitle)).Chosen() {
				m.OnOpenSessionPlugins()
			}
			if m.SessionPluginsActive && m.OnCloseSessionPlugins != nil && menu.Item(i18n.T(i18n.KeySessionPluginsStop)).Chosen() {
				m.OnCloseSessionPlugins()
			}
		}
		if m.OnOpenMarket != nil {
			if menu.Item(paneLabel("浏览更多插件", "Browse other plugins")).Chosen() {
				m.OnOpenMarket()
			}
		}
		if m.OnCreatePluginDraft != nil {
			if menu.Item(paneLabel("创建你的插件", "Create your plugin")).Chosen() {
				m.OnCreatePluginDraft("conversation")
			}
		}
		if m.OnOpenLauncher != nil && menu.Item(i18n.T(i18n.KeyLauncherOpen)).Chosen() {
			m.OnOpenLauncher()
		}
	})
}
