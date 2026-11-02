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
	// These immediate setters return the accepted stored value. On failure the
	// selector restores that value instead of changing the separate form draft.
	OnSetCloseToTray      func(bool) (bool, error)
	OnSetAutoStartDesktop func(bool) (bool, error)
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
	// 0.3.13 General: language, coding view, shortcuts, close, launch,
	// performance, session log, Harness version and auto-recovery. The
	// permission, send-behaviour, work-step and link rows moved to 对话.
	if d.Fields.Locale != nil {
		// The reference labels this row simply "语言" and keeps the
		// explanation in the select's own label.
		bodyPreferenceRow(c, "general-language", providerCopy("语言", "Language"), "", fg, muted, func() {
			lang := "中文"
			if *d.Fields.Locale == "en" {
				lang = "English"
			}
			if settingsPreferenceSelect(c, &lang, []string{"中文", "English"}).Disabled(d.OnSave == nil).Label(providerCopy("语言", "Language")).Changed() {
				if lang == "English" {
					*d.Fields.Locale = "en"
				} else {
					*d.Fields.Locale = "zh"
				}
			}
		})
	}
	bodyHostPreference(c, d.Host, "ui-settings", "enabled", providerCopy("显示代码工作视图", "Show coding view"), providerCopy("开启后显示轨迹、本轮代码差异，新对话中的 Agent 预设切换", "Show trajectory, this turn's code diff, and the Agent preset switch for new conversations."), fg, muted)
	if d.Shortcuts != nil {
		bodyShortcutPreference(c, d.Shortcuts, fg, muted)
	}
	if d.Fields.CloseToTray != nil {
		bodyPreferenceRow(c, "general-close", providerCopy("关闭窗口时", "When closing the window"), providerCopy("点标题栏关闭时：最小化到托盘，或退出并停止本机 Harness 服务。", "The title-bar close button can hide the app in the tray, or quit and stop the local Harness service."), fg, muted, func() {
			options := []string{providerCopy("最小化到托盘", "Minimize to tray"), providerCopy("直接退出", "Quit")}
			choice := options[1]
			if *d.Fields.CloseToTray {
				choice = options[0]
			}
			control := settingsPreferenceSelect(c, &choice, options).Disabled(d.OnSetCloseToTray == nil && d.OnSave == nil).Label(providerCopy("关闭窗口时", "When closing the window"))
			desktopImmediateBool(c, control, d.Fields.CloseToTray, &choice, options[0], d.OnSetCloseToTray)
		})
	}
	if d.AutoStartDesktop != nil {
		bodyPreferenceRow(c, "general-autostart", providerCopy("启动时", "On launch"), providerCopy("是：直接打开桌面端，启动器可从托盘右键「打开启动器」进入。\n否：先打开启动器，再手动启动桌面端。", "Yes: open the desktop directly; the launcher stays reachable from the tray menu.\nNo: open the launcher first and start the desktop manually."), fg, muted, func() {
			options := []string{providerCopy("直接启动桌面端", "Start desktop directly"), providerCopy("先打开启动器", "Open launcher first")}
			choice := options[1]
			if *d.AutoStartDesktop {
				choice = options[0]
			}
			control := settingsPreferenceSelect(c, &choice, options).Disabled(d.OnSetAutoStartDesktop == nil && d.OnSave == nil).Label(providerCopy("启动时", "On launch"))
			desktopImmediateBool(c, control, d.AutoStartDesktop, &choice, options[0], d.OnSetAutoStartDesktop)
		})
	}
	bodyHostPreference(c, d.Host, "ui-chat", "performanceUsage", providerCopy("性能与用量", "Performance and usage"), providerCopy("选择性能与用量信息展示的详细程度", "Choose how detailed the performance and usage information is"), fg, muted)
	bodyHostPreference(c, d.Host, "session-log-deepseek", "enabled", providerCopy("在使用官方模型 API 时上传 Session Log", "Upload Session Log when using the official model API"), providerCopy("帮助改进 DeepSeek 模型与产品", "Help improve DeepSeek models and products"), fg, muted)
	if d.Version != "" {
		// CurrentVersionRow is one full-width line, not a Setting-Cell's
		// fixed-width selector column: long build versions must remain readable.
		ui.Text(c.Key("general-version"), providerCopy("Harness 版本：", "Harness version: ")+d.Version).FillWidth().MinWidth(0).Padding(16, 0).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).Tooltip(d.Version).TextColor(fg)
	}
	harnessAutoRecoveryRow(c, d, fg, muted)
}

// desktopDiagnostics is the developer-facing remainder of the old General
// page: local runtime paths, the Electron/update probes, the verbose restart
// policy form and its explicit save. It is not part of the installed 鲸屿, so
// it lives behind a disclosure at the end of 关于.
func desktopDiagnostics(c *ui.Context, d *DesktopAccess, fg, muted ui.Color) {
	if d == nil {
		return
	}
	bodyLocalToggle(c, "general-login", i18n.T(i18n.KeySettingsDesktopOpenAtLogin), "", d.Fields.OpenAtLogin, d.OnSave == nil, fg, muted, nil)
	bodyDisclosure(c, "general-local-runtime", providerCopy("本地运行与诊断", "Local runtime and diagnostics"), func() {
		bodyHeading(c, i18n.T(i18n.KeySettingsDesktopTitle), i18n.T(i18n.KeySettingsDesktopHint), fg, muted)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if d.OnDetectElectron != nil {
				if ui.Button(c, i18n.T(i18n.KeySettingsDesktopDetectElectron)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
					d.OnDetectElectron()
				}
			}
			if d.OnCheckUpdates != nil {
				if ui.Button(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Disabled(d.UpdateBusy).Clicked() {
					d.OnCheckUpdates()
				}
			}
		})
		desktopRow(c, i18n.T(i18n.KeySettingsDesktopWorkspace), func() {
			if d.Fields.Workspace == nil {
				return
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, d.Fields.Workspace).Grow(1).Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13))
				if d.OnBrowseWorkspace != nil {
					if ui.Button(c, i18n.T(i18n.KeySettingsDesktopBrowseWorkspace)).Height(32).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Clicked() {
						d.OnBrowseWorkspace()
					}
				}
			})
		}, fg, muted)
		desktopRow(c, i18n.T(i18n.KeySettingsDesktopPort), func() {
			if d.Fields.Port == nil {
				return
			}
			ui.TextInput(c, d.Fields.Port).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13))
		}, fg, muted)
		harnessRestartPane(c, d, fg, muted)
	})
	// Theme/wallpaper → Appearance; locale/tray/login → Interface (ui-settings partition).
	save := ui.Button(c, i18n.T(i18n.KeySettingsSave)).Label("save-desktop-config").AlignSelf(ui.Start).Height(32).Padding(0, 14).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13))
	if d.OnSave == nil {
		save = save.Disabled(true)
	}
	if save.Clicked() && d.OnSave != nil {
		d.OnSave()
	}
}

// desktopImmediateBool keeps independent selectors separate from the local
// runtime form. A nil setter retains the handcrafted form's explicit-save path.
func desktopImmediateBool(c *ui.Context, control ui.Element, value *bool, choice *string, trueChoice string, save func(bool) (bool, error)) {
	failure := ui.Local(control, "immediate-save-error", func() string { return "" })
	if control.Changed() {
		next := *choice == trueChoice
		if save == nil {
			*value = next
		} else {
			accepted, err := save(next)
			*value = accepted
			*failure = ""
			if err != nil {
				*failure = err.Error()
			} else if accepted != next {
				*failure = providerCopy("设置未能保存，请重试。", "The setting was not saved; retry.")
			}
			if *failure != "" {
				c.AddToast(ui.Toast{ID: "desktop-immediate-preference", Title: *failure, Type: "error"})
			}
		}
	}
	if *failure != "" {
		ui.Text(c, *failure).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
	}
}

func desktopRow(c *ui.Context, label string, body func(), fg, muted ui.Color) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		body()
	})
	_ = muted
}
