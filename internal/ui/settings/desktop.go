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
		ui.Text(c, settingsActionText(d.LastAction)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	generalPermissionPreference(c, d.Host, fg, muted)
	if d.Fields.Locale != nil {
		// The reference General page labels this row simply "语言" and keeps the
		// explanation in the select's own label; the old description made the row
		// taller than the original's compact Setting-Cell.
		bodyPreferenceRow(c, "general-language", providerCopy("语言", "Language"), "", fg, muted, func() {
			lang := "中文"
			if *d.Fields.Locale == "en" {
				lang = "English"
			}
			if ui.Select(c, &lang, []string{"中文", "English"}).MinWidth(84).MaxWidth(140).Disabled(d.OnSave == nil).Label(providerCopy("语言", "Language")).Changed() {
				if lang == "English" {
					*d.Fields.Locale = "en"
				} else {
					*d.Fields.Locale = "zh"
				}
			}
		})
	}
	if d.Preferences != nil {
		bodyFontSizePreference(c, d.Preferences, fg, muted)
	}
	bodyHostPreference(c, d.Host, "ui-chat", "transcriptView", providerCopy("工作步骤展示", "Work step display"), providerCopy("选择希望看到多少工具调用细节", "Choose how much tool-call detail to show"), fg, muted)
	bodyHostPreference(c, d.Host, "ui-chat", "collapseTiming", providerCopy("工作步骤收起时机", "Collapse completed work steps"), providerCopy("选择何时自动收起工作步骤", "Choose when completed work steps collapse"), fg, muted)
	bodyHostPreference(c, d.Host, "ui-settings", "enabled", providerCopy("显示代码工作视图", "Show the code work view"), providerCopy("开启后显示轨迹、本轮代码差异，新对话中的 Agent 预设切换。", "Show trajectory, this turn's code diff, and the Agent preset switch for new conversations."), fg, muted)
	if d.Shortcuts != nil {
		bodyShortcutPreference(c, d.Shortcuts, fg, muted)
	}
	bodyHostPreference(c, d.Host, "ui-chat", "linkOpening", providerCopy("网页链接默认打开方式", "Default link opening"), providerCopy("对话中网页链接的打开位置", "Where links in the conversation open"), fg, muted)
	bodyHostPreference(c, d.Host, "ui-conversation", "busyEnter", providerCopy("繁忙时的发送行为", "Sending while busy"), providerCopy("智能体运行时 Enter 和发送按钮的行为；Cmd/Ctrl+Enter 使用另一行为。", "What Enter and Send do while the agent runs; Cmd/Ctrl+Enter uses the other action."), fg, muted)
	if d.Fields.CloseToTray != nil {
		bodyPreferenceRow(c, "general-close", providerCopy("关闭窗口时", "Closing the window"), providerCopy("点标题栏关闭时：最小化到托盘，或退出并停止本机 Harness 服务。", "On the title-bar close button: minimize to the tray, or quit and stop the local Harness service."), fg, muted, func() {
			options := []string{providerCopy("最小化到托盘", "Minimize to tray"), providerCopy("直接退出", "Quit")}
			choice := options[1]
			if *d.Fields.CloseToTray {
				choice = options[0]
			}
			if ui.Select(c, &choice, options).MinWidth(84).MaxWidth(140).Disabled(d.OnSave == nil).Label(providerCopy("关闭窗口时", "Closing the window")).Changed() {
				*d.Fields.CloseToTray = choice == options[0]
			}
		})
	}
	bodyLocalToggle(c, "general-login", i18n.T(i18n.KeySettingsDesktopOpenAtLogin), "", d.Fields.OpenAtLogin, d.OnSave == nil, fg, muted, nil)
	if d.AutoStartDesktop != nil {
		bodyPreferenceRow(c, "general-autostart", i18n.T(i18n.KeySettingsInterfaceAutoStart), providerCopy("是：直接打开桌面端；否：先打开启动器，再手动启动桌面端。", "Yes: open the desktop directly; No: open the launcher first and start the desktop manually."), fg, muted, func() {
			options := []string{providerCopy("直接启动桌面端", "Open the desktop"), providerCopy("先打开启动器", "Open the launcher")}
			choice := options[1]
			if *d.AutoStartDesktop {
				choice = options[0]
			}
			if ui.Select(c, &choice, options).MinWidth(84).MaxWidth(140).Disabled(d.OnSave == nil).Label(i18n.T(i18n.KeySettingsInterfaceAutoStart)).Changed() {
				*d.AutoStartDesktop = choice == options[0]
			}
		})
	}
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
