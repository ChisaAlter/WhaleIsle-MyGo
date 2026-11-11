package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/credentials"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"whaleisle/internal/ui/i18n"
)

// AboutAccess is the dedicated About settings page.
type AboutAccess struct {
	Loading, UpdateBusy, HomeBusy       bool
	Err, Status, UpdateStatus, HomePath string
	OnInstallUpdate                     func()
	OnRevealHome                        func() error
	ReadCredentialStorage               func() credentials.StorageStatus
	OnOpenLink                          func(string) error
	AppName                             string
	Version                             string
	Identifier                          string
	UserData                            string
	VendorRoot                          string
	OnRevealUserData                    func()
	OnRevealLogs                        func()
	OnRevealVendor                      func()
	OnCheckUpdates                      func()
	OnCopyUserData                      func()
	OnCopyVersion                       func()
	AnalyticsEnabled                    *bool
	OnRefreshAnalytics                  func()
	UpdatePhase                         string
	UpdateVersion                       string
	UpdatePercent                       float64
	OnDownloadUpdate                    func()
	OnResumeUpdate                      func()
	OnOpenUpdateCenter                  func()
}

func aboutUpdatePhaseSummary(a *AboutAccess) string {
	if a == nil {
		return ""
	}
	switch a.UpdatePhase {
	case "checking":
		return providerCopy("正在检测更新…", "Checking for updates…")
	case "current":
		if latest := strings.TrimSpace(a.UpdateVersion); latest != "" {
			return providerCopy("已是最新版本 "+latest, "You are on the latest version "+latest)
		}
		return providerCopy("已是最新版本", "You are on the latest version")
	case "available":
		if latest := strings.TrimSpace(a.UpdateVersion); latest != "" {
			return providerCopy("发现新版本 "+latest, "New version "+latest+" is available")
		}
		return providerCopy("发现新版本", "New version available")
	case "downloading":
		return fmt.Sprintf(providerCopy("正在下载 %.0f%%", "Downloading %.0f%%"), a.UpdatePercent)
	case "paused":
		if a.UpdatePercent > 0 {
			return fmt.Sprintf("%s %.0f%%", providerCopy("下载已暂停", "Download paused"), a.UpdatePercent)
		}
		return providerCopy("下载已暂停", "Download paused")
	case "verifying":
		return providerCopy("正在校验更新包…", "Verifying update…")
	case "ready":
		return providerCopy("更新已就绪，重启后完成安装", "Update ready to install")
	case "installing":
		return providerCopy("已启动安装程序，请按向导完成安装。", "Installer launched. Follow the wizard to finish.")
	case "error":
		return providerCopy("检测或安装失败", "Update failed")
	default:
		return ""
	}
}

func aboutUpdateCenterSubtitle(a *AboutAccess) string {
	if a == nil {
		return ""
	}
	version := strings.TrimSpace(a.UpdateVersion)
	if version == "" {
		version = strings.TrimSpace(a.Version)
	}
	current := strings.TrimSpace(a.Version)
	percent := fmt.Sprintf("%.0f", a.UpdatePercent)
	switch a.UpdatePhase {
	case "idle":
		if current != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterIdle, "current", current)
		}
	case "current":
		if current != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterCurrent, "current", current)
		}
	case "available":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterAvailable, "version", version)
		}
	case "downloading":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterDownloading, "version", version, "percent", percent)
		}
	case "paused":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterPaused, "version", version, "percent", percent)
		}
	case "verifying":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterVerifying, "version", version)
		}
	case "ready":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterReady, "version", version)
		}
	case "installing":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterInstalling, "version", version)
		}
	case "error":
		msg := strings.TrimSpace(a.UpdateStatus)
		if msg == "" {
			msg = strings.TrimSpace(a.Err)
		}
		if msg != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterError, "message", msg)
		}
	}
	return ""
}

func aboutUpdateCenterErrorAlt(a *AboutAccess) string {
	if a == nil || a.UpdatePhase != "error" {
		return ""
	}
	msg := strings.TrimSpace(a.UpdateStatus)
	if msg == "" {
		msg = strings.TrimSpace(a.Err)
	}
	if msg == "" {
		return ""
	}
	return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterErrorAlt, "message", msg)
}

func aboutPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.About == nil {
		return
	}
	a := m.About
	accent := chrome.Color(fonttheme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)
	anchor := ui.Column(c.Key("about-heading")).FillWidth().MaxWidth(560).MinWidth(0).Gap(12)
	storage := ui.Local(anchor, "metadata", func() credentials.StorageStatus {
		if a.ReadCredentialStorage != nil {
			return a.ReadCredentialStorage()
		}
		return credentials.StorageStatus{Mode: "unknown"}
	})
	linkError := ui.Local(anchor, "error", func() string { return "" })
	homeResult := ui.Local(anchor, "home-result", func() string { return "" })
	meta := func(text string) {
		ui.Text(c, text).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
	}
	link := func(link aboutLink) {
		button := ui.Button(c, link.Label).AlignSelf(ui.Start).MinWidth(0).Height(fonttheme.InterfaceSize(21)).Padding(0).Radius(0).BorderWidth(0).Background(ui.Transparent).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(21)).TextColor(accent).Tooltip(link.URL).Disabled(a.OnOpenLink == nil)
		if button.Hovered() {
			button.Underline()
		}
		if button.Clicked() {
			*linkError = ""
			if err := a.OnOpenLink(link.URL); err != nil {
				*linkError = err.Error()
			}
		}
	}
	anchor.Children(func() {
		// Keep the primary update status and next action directly under the
		// desktop version. Only local diagnostics belong in the disclosure.
		ui.Text(c, i18n.T(KeyPass4AboutTitle)).FillWidth().FontSize(fonttheme.InterfaceSize(18)).FixedLineHeight(fonttheme.InterfaceSize(24)).FontWeight(600).TextColor(fg)
		ui.Text(c, providerCopy("本桌面应用与官方 Harness。", "This desktop app and official Harness.")).FillWidth().FontSize(fonttheme.InterfaceSize(13)).FixedLineHeight(fonttheme.InterfaceSize(19.5)).TextColor(muted)
		settingsNotice(c, "about:118", "error", a.Err != "", func() string { return a.Err })
		links := aboutLinks()
		ui.Column(c).FillWidth().MinWidth(0).Gap(14).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, emptyDash(a.AppName)).FillWidth().FontSize(fonttheme.InterfaceSize(15)).FixedLineHeight(fonttheme.InterfaceSize(21)).FontWeight(600).TextColor(fg)
				meta(providerCopy("桌面版本 ", "Desktop version ") + emptyDash(a.Version))
				aboutUpdateRow(c, a, fg, muted)
				meta("Based on DeepSeek Harness")
				meta(providerCopy("凭据存储：", "Credential storage: ") + credentialStorageSummary(*storage))
				if a.HomePath != "" {
					meta(providerCopy("运行目录 ", "Runtime folder ") + a.HomePath)
				}
				if a.OnRevealHome != nil {
					colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
					ui.Row(c).FillWidth().Margin(10, 0, 8).Children(func() {
						folder := widgets.Button(c, colors, i18n.T(KeyPass4OpenRuntime),
							widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: a.HomeBusy})
						folder.AlignSelf(ui.Start)
						if folder.Clicked() {
							if err := a.OnRevealHome(); err != nil {
								*homeResult = err.Error()
							} else {
								*homeResult = ""
							}
						}
					})
				}
				if *homeResult != "" {
					meta(*homeResult)
				}
				link(links[0])
			})
			link(links[1])
		})
		settingsNotice(c, "about:154", "error", *linkError != "", func() string { return *linkError })
		settingsNotice(c, "about:157", "", a.Status != "", func() string { return a.Status })
		bodyDisclosure(c, "about-local-details", providerCopy("应用数据与诊断", "Application data and diagnostics"), func() {
			ui.Text(c, credentialStorageDescription(*storage)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			for _, row := range []struct{ id, label, value string }{{"about-detail-id", i18n.T(i18n.KeySettingsAboutID), a.Identifier}, {"about-detail-user-data", i18n.T(i18n.KeySettingsAboutUserData), a.UserData}, {"about-detail-vendor", i18n.T(i18n.KeySettingsAboutVendor), a.VendorRoot}} {
				if row.value != "" {
					element := bodyPreferenceRow(c, "about-detail-"+row.label, row.label, "", fg, muted, func() {
						ui.Text(c, row.value).FillWidth().SingleLine().Tooltip(row.value).Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					})
					settingsSearchTarget(c, row.id, row.label, element)
				}
			}
			if a.AnalyticsEnabled != nil {
				value := i18n.T(i18n.KeySettingsAboutAnalyticsOff)
				if *a.AnalyticsEnabled {
					value = i18n.T(i18n.KeySettingsAboutAnalyticsOn)
				}
				ui.Text(c, i18n.T(i18n.KeySettingsAboutAnalytics)+": "+value).FillWidth().TextColor(muted)
			}
			if a.ReadCredentialStorage != nil && ui.Button(c, providerCopy("刷新凭据存储状态", "Refresh credential storage status")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Clicked() {
				*storage = a.ReadCredentialStorage()
			}
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				for _, action := range []struct {
					label string
					run   func()
				}{{i18n.T(i18n.KeySettingsAboutRevealData), a.OnRevealUserData}, {i18n.T(i18n.KeySettingsAboutRevealLogs), a.OnRevealLogs}, {i18n.T(i18n.KeySettingsAboutRevealVendor), a.OnRevealVendor}, {i18n.T(i18n.KeySettingsAboutCopyData), a.OnCopyUserData}, {i18n.T(i18n.KeySettingsRefresh), a.OnRefreshAnalytics}} {
					if action.run != nil && ui.Button(c, action.label).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Clicked() {
						action.run()
					}
				}
			})
			if hasDesktopDiagnostics(m.Desktop) {
				desktopDiagnostics(c, m.Desktop, fg, muted)
			}
			if m.Store != nil {
				advancedNavigation(c, m, m.Store.Snapshot(), fg, muted)
			}
		})
	})
}

// aboutUpdateRow is the primary version/update surface immediately below the version.
func aboutUpdateRow(c *ui.Context, a *AboutAccess, fg, muted ui.Color) {
	ui.Row(c).FillWidth().Wrap().Gap(8).Margin(10, 0, 8).AlignItems(ui.Center).Children(func() {
		// The version is data, not a choice: it paints as read-only text
		// instead of a disabled picker with one option.
		ui.Text(c, emptyDash(a.Version)).Label(i18n.T(i18n.KeySettingsAboutCopyVersion)).
			Height(28).Padding(0, 10).Radius(8).Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .35)).
			FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Tooltip(a.Version)
		installLabel := i18n.T(KeyPass4OnlineInstall)
		// The badge keeps its near-black fill even when the installer is
		// absent; only the label fades, so it still reads as a badge.
		badgeInk := ui.RGBA(255, 255, 255, .95)
		enabled := a.OnInstallUpdate != nil && !a.UpdateBusy
		if !enabled {
			badgeInk = ui.RGBA(255, 255, 255, .62)
		}
		install := ui.Button(c, "").Label(installLabel).Height(28).Padding(0, 12).Radius(8).
			Background(pass4InstallBadgeFill).AlignSelf(ui.Start).Disabled(!enabled)
		if !enabled {
			install = install.Tooltip(i18n.T(KeyPass4InstallUnavailable))
		}
		install.Children(func() {
			ui.Text(c, installLabel).FontSize(fonttheme.InterfaceSize(12)).TextColor(badgeInk)
		})
		if install.Clicked() && enabled {
			a.OnInstallUpdate()
		}
		if a.OnCheckUpdates != nil && ui.Button(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates)).Height(28).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12)).Disabled(a.UpdateBusy).Clicked() {
			a.OnCheckUpdates()
		}
		if a.UpdatePhase == "available" && a.OnDownloadUpdate != nil && ui.Button(c, providerCopy("下载更新", "Download update")).Height(28).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12)).Disabled(a.UpdateBusy).Clicked() {
			a.OnDownloadUpdate()
		}
		if (a.UpdatePhase == "paused" || a.UpdatePhase == "error") && a.OnResumeUpdate != nil && ui.Button(c, providerCopy("继续下载", "Resume download")).Height(28).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12)).Disabled(a.UpdateBusy).Clicked() {
			a.OnResumeUpdate()
		}
		if a.UpdatePhase == "ready" && a.OnInstallUpdate != nil && ui.Button(c, providerCopy("安装并重启", "Install and restart")).Height(28).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12)).Disabled(a.UpdateBusy).Clicked() {
			a.OnInstallUpdate()
		}
		if a.OnOpenUpdateCenter != nil && ui.Button(c, providerCopy("打开更新中心", "Open Update Center")).Height(28).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
			a.OnOpenUpdateCenter()
		}
		if a.OnOpenUpdateCenter != nil && ui.Button(c, i18n.T(i18n.KeySettingsParityAboutCenterOpen)).Height(28).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
			a.OnOpenUpdateCenter()
		}
	})
	if summary := aboutUpdatePhaseSummary(a); summary != "" && summary != a.UpdateStatus && !(a.UpdateBusy && a.UpdatePhase == "checking") {
		ui.Text(c, summary).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if center := aboutUpdateCenterSubtitle(a); center != "" && center != aboutUpdatePhaseSummary(a) && !(a.UpdateBusy && a.UpdatePhase == "checking") {
		ui.Text(c, center).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if altErr := aboutUpdateCenterErrorAlt(a); altErr != "" && altErr != aboutUpdateCenterSubtitle(a) && !(a.UpdateBusy && a.UpdatePhase == "checking") {
		ui.Text(c, altErr).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if a.UpdateStatus != "" {
		ui.Text(c, a.UpdateStatus).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
}

// credentialStorageSummary is the one-line value after "凭据存储：".
func credentialStorageSummary(s credentials.StorageStatus) string {
	switch s.Mode {
	case "encrypted":
		return providerCopy("系统钥匙串加密（DPAPI）", "Encrypted by the system keychain (DPAPI)")
	case "plaintext":
		return providerCopy("明文文件（不安全）", "Plaintext file (unsafe)")
	case "missing":
		return providerCopy("尚未创建，保存时使用系统钥匙串加密", "Not created yet; saved credentials use the system keychain")
	default:
		return providerCopy("无法确认", "Unknown")
	}
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

type aboutLink struct{ Label, URL string }

func aboutLinks() []aboutLink {
	return []aboutLink{
		{i18n.T(KeyPass4GitHubReleases), "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases"},
		{providerCopy("官方 Harness", "Official Harness"), "https://github.com/deepseek-ai/deepseek-harness"},
	}
}
func credentialStorageDescription(s credentials.StorageStatus) string {
	text := ""
	switch s.Mode {
	case "plaintext":
		text = providerCopy("警告：现有凭据文件为明文，能访问该文件的人可读取凭据。", "Warning: the existing credential file is plaintext. Anyone with file access can read its credentials.")
	case "encrypted":
		text = providerCopy("已检测到加密封装；尚未验证系统密钥能否解锁。", "Encrypted envelope detected; OS key unlock has not been verified.")
	case "missing":
		text = providerCopy("尚未创建凭据文件。保存需要可用的 Windows DPAPI 密钥；不会回退为明文。", "No credential file exists. Saving requires an available Windows DPAPI key; there is no plaintext fallback.")
	default:
		text = providerCopy("无法确认当前凭据存储状态。", "Current credential storage status is unknown.")
	}
	if !s.SupportedPlatform {
		text += providerCopy(" 当前平台不支持此 DPAPI 写入后端。", " This platform does not support this DPAPI write backend.")
	}
	return text
}
