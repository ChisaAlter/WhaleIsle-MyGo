package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"

	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
)

// InterfaceAccess is the dedicated Interface settings page (locale + chrome prefs).
type InterfaceAccess struct {
	Host                                  *BodySettingsAccess
	DshbotEnabled, RemoteWorkspaceEnabled *bool
	OnSetDshbot, OnSetRemoteWorkspace     func(bool)
	Shortcuts                             *shortcuts.Editor
	Locale                                *string
	CloseToTray                           *bool
	OpenAtLogin                           *bool
	// AutoStartDesktop starts harness after boot when workspace is set (launcher toggle).
	AutoStartDesktop *bool
	// ShowViewTabs mirrors Electron conversation settings.viewTabs (inverted HideChatViewTabs).
	ShowViewTabs     *bool
	ShowArchivedList *bool
	// AutoAllowTools mirrors Electron tool auto-allow (interaction store).
	AutoAllowTools *bool
	OnSave         func()
	LastAction     string
	Err            string
}

func interfacePane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Interface == nil {
		return
	}
	a := m.Interface
	bodyHostState(c, a.Host, muted)
	for _, field := range []struct{ ns, key, zh, en string }{
		{"ui-git", "titlebarGit", "标题栏 Git", "Titlebar Git"},
		{"ui-titlebar", "terminalToggle", "终端入口", "Terminal toggle"},
		{"ui-titlebar", "surfacesToggle", "面板入口", "Panel toggles"},
		{"ui-conversation", "composerResize", "调整输入框大小", "Resize composer"},
		{"ui-conversation", "statsLine", "会话统计", "Session statistics"},
		{"ui-conversation", "sessionCost", "会话费用", "Session cost"},
		{"ui-conversation", "officialPeakValley", "官方峰谷时段", "Official peak and valley hours"},
		{"ui-conversation", "viewTabs", "会话视图标签", "Conversation view tabs"},
		{"ui-conversation", "nextPrompt", "下一条消息建议", "Next prompt suggestions"},
	} {
		if field.key == "composerResize" {
			bodyEffectPreference(c, a.Host, false, fg, muted)
		}
		bodyHostPreference(c, a.Host, field.ns, field.key, providerCopy(field.zh, field.en), "", fg, muted)
	}
	bodyToggleRowBadge(c, "interface-dshbot", providerCopy("机器人（Bots）", "Bots"), providerCopy("测试中", "Beta"), providerCopy("侧栏启用机器人联系人与群聊。该功能仍在测试，可能不稳定；切换后会自动重启 Harness。", "Turns on the sidebar bot contact and group chat. This feature is still in beta and may be unstable; switching restarts Harness."), a.DshbotEnabled, a.OnSetDshbot == nil, fg, muted, a.OnSetDshbot)
	bodyLocalToggle(c, "interface-remote-workspace", providerCopy("远程工作区", "Remote workspaces"), providerCopy("启用远程工作区的目录与连接入口。", "Enable remote workspace directory and connection entries."), a.RemoteWorkspaceEnabled, a.OnSetRemoteWorkspace == nil, fg, muted, a.OnSetRemoteWorkspace)
	bodyLocalToggle(c, "interface-archive", providerCopy("显示归档列表", "Show archived list"), "", a.ShowArchivedList, a.OnSave == nil, fg, muted, nil)
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	bodyDisclosure(c, "interface-local-preferences", providerCopy("本地偏好与快捷键", "Local preferences and shortcuts"), func() {
		desktopRow(c, i18n.T(i18n.KeySettingsDesktopLocale), func() {
			if a.Locale == nil {
				return
			}
			ui.Select(c, a.Locale, []string{"zh", "en"}).FillWidth()
		}, fg, muted)
		desktopRow(c, i18n.T(i18n.KeySettingsDesktopCloseToTray), func() {
			if a.CloseToTray == nil {
				return
			}
			ui.Switch(c, a.CloseToTray).Label(i18n.T(i18n.KeySettingsDesktopCloseToTray))
		}, fg, muted)
		desktopRow(c, i18n.T(i18n.KeySettingsDesktopOpenAtLogin), func() {
			if a.OpenAtLogin == nil {
				return
			}
			ui.Switch(c, a.OpenAtLogin).Label(i18n.T(i18n.KeySettingsDesktopOpenAtLogin))
		}, fg, muted)
		desktopRow(c, i18n.T(i18n.KeySettingsInterfaceAutoStart), func() {
			if a.AutoStartDesktop == nil {
				return
			}
			ui.Switch(c, a.AutoStartDesktop).Label(i18n.T(i18n.KeySettingsInterfaceAutoStart))
		}, fg, muted)
		ui.Text(c, i18n.T(i18n.KeySettingsInterfaceAutoStartDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		desktopRow(c, i18n.T(i18n.KeySettingsInterfaceViewTabs), func() {
			if a.ShowViewTabs == nil {
				return
			}
			ui.Switch(c, a.ShowViewTabs).Label(i18n.T(i18n.KeySettingsInterfaceViewTabs))
		}, fg, muted)
		ui.Text(c, i18n.T(i18n.KeySettingsInterfaceViewTabsDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		desktopRow(c, i18n.T(i18n.KeySettingsInterfaceAutoAllow), func() {
			if a.AutoAllowTools == nil {
				return
			}
			ui.Switch(c, a.AutoAllowTools).Label(i18n.T(i18n.KeySettingsInterfaceAutoAllow))
		}, fg, muted)
		ui.Text(c, i18n.T(i18n.KeySettingsInterfaceAutoAllowDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		if a.OnSave != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
			a.OnSave()
		}
		shortcutsPane(c, a.Shortcuts, fg, muted)
	})
	if a.ShowArchivedList != nil && a.OnSave != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Label("save-interface-preferences").Clicked() {
		a.OnSave()
	}
}
