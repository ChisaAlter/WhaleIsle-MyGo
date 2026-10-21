package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"

	"whaleisle/internal/config"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
)

// DesktopFields is the editable subset of internal/config shown in the 桌面 section.
type DesktopFields struct {
	HarnessAutoRestart        *bool
	HarnessRestartMaxAttempts *string
	HarnessRestartBaseDelayMs *string
	Workspace                 *string
	Port                      *string
	Theme                     *string
	Locale                    *string
	CloseToTray               *bool
	OpenAtLogin               *bool
	WallpaperPath             *string
	// WallpaperChoices are absolute image paths for the gallery Select
	// (first option is empty = none).
	WallpaperChoices []string
}

// DesktopAccess reads and writes local config.json via callbacks from main.
type DesktopAccess struct {
	Host             *BodySettingsAccess
	Shortcuts        *shortcuts.Editor
	Preferences      *theme.Editor
	Version          string
	AutoStartDesktop *bool
	// RestartPolicy is the committed snapshot, separate from the editable draft.
	RestartPolicy     *config.RestartPreferences
	Fields            DesktopFields
	OnSave            func()
	OnDetectElectron  func()
	OnCheckUpdates    func()
	UpdateBusy        bool
	OnBrowseWorkspace func()
	LastAction        string
	Err               string
}

func desktopPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Desktop == nil {
		return
	}
	d := m.Desktop
	bodyHostState(c, d.Host, muted)
	if d.Err != "" {
		ui.Text(c, d.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg)
	}
	if d.LastAction != "" {
		ui.Text(c, d.LastAction).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	generalPermissionPreference(c, d.Host, fg, muted)
	if d.Fields.Locale != nil {
		bodyPreferenceRow(c, "general-language", i18n.T(i18n.KeySettingsDesktopLocale), providerCopy("选择界面使用的语言。", "Choose the interface language."), fg, muted, func() {
			ui.Select(c, d.Fields.Locale, []string{"zh", "en"}).FillWidth().Disabled(d.OnSave == nil).Label(i18n.T(i18n.KeySettingsDesktopLocale))
		})
	}
	if d.Preferences != nil {
		bodyFontSizePreference(c, d.Preferences, fg, muted)
	}
	bodyHostPreference(c, d.Host, "ui-chat", "transcriptView", providerCopy("工作详情", "Work details"), providerCopy("选择任务过程展示的详细程度。", "Choose how much of the task's work is shown."), fg, muted)
	bodyHostPreference(c, d.Host, "ui-chat", "collapseTiming", providerCopy("工作详情收起时机", "Collapse completed work"), "", fg, muted)
	bodyHostPreference(c, d.Host, "ui-settings", "enabled", providerCopy("显示代码工作视图", "Show the code work view"), providerCopy("开启后显示轨迹、本轮代码差异，新对话中的 Agent 预设切换。", "Show trajectory, this turn's code diff, and the Agent preset switch for new conversations."), fg, muted)
	if d.Shortcuts != nil {
		bodyDisclosure(c, "general-shortcuts", providerCopy("键盘快捷键", "Keyboard shortcuts"), func() { shortcutsPane(c, d.Shortcuts, fg, muted) })
	}
	bodyHostPreference(c, d.Host, "ui-chat", "linkOpening", providerCopy("链接打开方式", "Open links in"), "", fg, muted)
	bodyHostPreference(c, d.Host, "ui-conversation", "busyEnter", providerCopy("繁忙时的发送行为", "Sending while busy"), providerCopy("智能体运行时 Enter 和发送按钮的行为；Cmd/Ctrl+Enter 使用另一行为。", "What Enter and Send do while the agent runs; Cmd/Ctrl+Enter uses the other action."), fg, muted)
	bodyLocalToggle(c, "general-close", providerCopy("关闭窗口时最小化到托盘", "Minimize to tray on close"), providerCopy("关闭时隐藏到托盘；关闭此开关则直接退出并停止本机 Harness 服务。", "Hide to the tray on close; turn this off to quit and stop the local Harness service."), d.Fields.CloseToTray, d.OnSave == nil, fg, muted, nil)
	bodyLocalToggle(c, "general-login", i18n.T(i18n.KeySettingsDesktopOpenAtLogin), "", d.Fields.OpenAtLogin, d.OnSave == nil, fg, muted, nil)
	bodyLocalToggle(c, "general-autostart", i18n.T(i18n.KeySettingsInterfaceAutoStart), i18n.T(i18n.KeySettingsInterfaceAutoStartDesc), d.AutoStartDesktop, d.OnSave == nil, fg, muted, nil)
	generalCustomInstructions(c, d.Host, fg, muted)
	bodyHostPreference(c, d.Host, "visual-replies", "enabled", providerCopy("HTML 视觉回复", "HTML visual replies"), "", fg, muted)
	bodyHostPreference(c, d.Host, "ui-chat", "performanceUsage", providerCopy("性能与用量", "Performance and usage"), "", fg, muted)
	bodyHostPreference(c, d.Host, "session-log-deepseek", "enabled", providerCopy("会话日志上传", "Upload session logs"), "", fg, muted)
	if d.Version != "" {
		// CurrentVersionRow is one full-width line, not a Setting-Cell's
		// fixed-width selector column: long build versions must remain readable.
		ui.Text(c.Key("general-version"), providerCopy("Harness 版本：", "Harness version: ")+d.Version).FillWidth().MinWidth(0).Padding(16, 0).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).Tooltip(d.Version).TextColor(fg)
	}
	bodyDisclosure(c, "general-local-runtime", providerCopy("本地运行与诊断", "Local runtime and diagnostics"), func() {
		bodyHeading(c, i18n.T(i18n.KeySettingsDesktopTitle), i18n.T(i18n.KeySettingsDesktopHint), fg, muted)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if d.OnDetectElectron != nil {
				if ui.Button(c, i18n.T(i18n.KeySettingsDesktopDetectElectron)).Clicked() {
					d.OnDetectElectron()
				}
			}
			if d.OnCheckUpdates != nil {
				if ui.Button(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates)).Disabled(d.UpdateBusy).Clicked() {
					d.OnCheckUpdates()
				}
			}
		})
		desktopRow(c, i18n.T(i18n.KeySettingsDesktopWorkspace), func() {
			if d.Fields.Workspace == nil {
				return
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, d.Fields.Workspace).Grow(1)
				if d.OnBrowseWorkspace != nil {
					if ui.Button(c, i18n.T(i18n.KeySettingsDesktopBrowseWorkspace)).FontSize(theme.InterfaceSize(11)).Clicked() {
						d.OnBrowseWorkspace()
					}
				}
			})
		}, fg, muted)
		desktopRow(c, i18n.T(i18n.KeySettingsDesktopPort), func() {
			if d.Fields.Port == nil {
				return
			}
			ui.TextInput(c, d.Fields.Port).FillWidth()
		}, fg, muted)
		harnessRestartPane(c, d, fg, muted)
	})
	// Theme/wallpaper → Appearance; locale/tray/login → Interface (ui-settings partition).
	save := ui.Button(c, i18n.T(i18n.KeySettingsSave)).Label("save-desktop-config")
	if d.OnSave == nil {
		save = save.Disabled(true)
	}
	if save.Clicked() && d.OnSave != nil {
		d.OnSave()
	}
}

func desktopRow(c *ui.Context, label string, body func(), fg, muted ui.Color) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		body()
	})
	_ = muted
}
