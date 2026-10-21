package settings

import (
	"whaleisle/internal/ui/widgets"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/credentials"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"

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
}

func aboutPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.About == nil {
		return
	}
	a := m.About
	accent := chrome.Color(fonttheme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)
	anchor := ui.Column(c.Key("about-heading")).FillWidth().MaxWidth(520).MinWidth(0).Gap(12)
	storage := ui.Local(anchor, "metadata", func() credentials.StorageStatus {
		if a.ReadCredentialStorage != nil {
			return a.ReadCredentialStorage()
		}
		return credentials.StorageStatus{Mode: "unknown"}
	})
	linkError := ui.Local(anchor, "error", func() string { return "" })
	homeResult := ui.Local(anchor, "home-result", func() string { return "" })
	anchor.Children(func() {
		ui.Text(c, i18n.T(KeyPass4AboutTitle)).FillWidth().FontSize(fonttheme.InterfaceSize(18)).FontWeight(600).TextColor(fg)
		ui.Text(c, providerCopy("本桌面应用与官方 Harness。", "This desktop app and the official Harness.")).FillWidth().FontSize(fonttheme.InterfaceSize(13)).FixedLineHeight(fonttheme.InterfaceSize(19.5)).TextColor(muted)
		if a.Loading {
			ui.Text(c, providerCopy("正在读取应用信息…", "Loading application information…")).FillWidth().TextColor(muted)
		}
		if a.Err != "" {
			ui.Text(c, a.Err).FillWidth().TextColor(fg)
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(4).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).Children(func() {
			ui.Text(c, emptyDash(a.AppName)).FillWidth().FontSize(fonttheme.InterfaceSize(15)).FixedLineHeight(fonttheme.InterfaceSize(21)).FontWeight(600).TextColor(fg)
			ui.Text(c, providerCopy("桌面版本 ", "Desktop version ")+emptyDash(a.Version)).FillWidth().Tooltip(a.Version).TextColor(muted)
			ui.Text(c, "Based on DeepSeek Harness").FillWidth().TextColor(muted)
			// Version row: the installed version as a one-row picker, then the
			// installer badge. Nothing is faked: without an installer the badge
			// explains itself instead of pretending to start a download.
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
					Background(pass4InstallBadgeFill).AlignSelf(ui.Start)
				if !enabled {
					install = install.Tooltip(i18n.T(KeyPass4InstallUnavailable))
				}
				install.Children(func() {
					ui.Text(c, installLabel).FontSize(fonttheme.InterfaceSize(12)).TextColor(badgeInk)
				})
				if install.Clicked() && a.OnInstallUpdate != nil {
					a.OnInstallUpdate()
				}
				if a.OnCheckUpdates != nil && ui.Button(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates)).Disabled(a.UpdateBusy).Clicked() {
					a.OnCheckUpdates()
				}
			})
			if a.UpdateBusy {
				ui.Text(c, providerCopy("正在检测更新…", "Checking for updates…")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
			if a.UpdateStatus != "" {
				ui.Text(c, a.UpdateStatus).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
			ui.Text(c, credentialStorageDescription(*storage)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			if storage.Error != "" {
				ui.Text(c, storage.Error).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
			}
			if a.HomePath != "" {
				ui.Text(c, providerCopy("运行目录 ", "Runtime directory ")+a.HomePath).FillWidth().Tooltip(a.HomePath).TextColor(muted)
			}
			if a.OnRevealHome != nil {
				colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
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
			}
			if *homeResult != "" {
				ui.Text(c, *homeResult).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
		})
		for _, link := range aboutLinks() {
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
		if *linkError != "" {
			ui.Text(c, *linkError).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		if a.Status != "" {
			ui.Text(c, a.Status).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		bodyDisclosure(c, "about-local-details", providerCopy("应用数据与诊断", "Application data and diagnostics"), func() {
			for _, row := range []struct{ label, value string }{{i18n.T(i18n.KeySettingsAboutID), a.Identifier}, {i18n.T(i18n.KeySettingsAboutUserData), a.UserData}, {i18n.T(i18n.KeySettingsAboutVendor), a.VendorRoot}} {
				if row.value != "" {
					bodyPreferenceRow(c, "about-detail-"+row.label, row.label, "", fg, muted, func() {
						ui.Text(c, row.value).FillWidth().SingleLine().Tooltip(row.value).Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					})
				}
			}
			if a.AnalyticsEnabled != nil {
				value := i18n.T(i18n.KeySettingsAboutAnalyticsOff)
				if *a.AnalyticsEnabled {
					value = i18n.T(i18n.KeySettingsAboutAnalyticsOn)
				}
				ui.Text(c, i18n.T(i18n.KeySettingsAboutAnalytics)+": "+value).FillWidth().TextColor(muted)
			}
			if a.ReadCredentialStorage != nil && ui.Button(c, providerCopy("刷新凭据存储状态", "Refresh credential storage status")).Clicked() {
				*storage = a.ReadCredentialStorage()
			}
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				for _, action := range []struct {
					label string
					run   func()
				}{{i18n.T(i18n.KeySettingsAboutRevealData), a.OnRevealUserData}, {i18n.T(i18n.KeySettingsAboutRevealLogs), a.OnRevealLogs}, {i18n.T(i18n.KeySettingsAboutRevealVendor), a.OnRevealVendor}, {i18n.T(i18n.KeySettingsAboutCopyData), a.OnCopyUserData}, {i18n.T(i18n.KeySettingsRefresh), a.OnRefreshAnalytics}} {
					if action.run != nil && ui.Button(c, action.label).Clicked() {
						action.run()
					}
				}
			})
		})
	})
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