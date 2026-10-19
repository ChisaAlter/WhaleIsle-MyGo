package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/credentials"
	fonttheme "whaleisle/internal/ui/theme"

	"whaleisle/internal/ui/i18n"
)

// AboutAccess is the dedicated About settings page.
type AboutAccess struct {
	Loading, UpdateBusy, HomeBusy       bool
	Err, Status, UpdateStatus, HomePath string
	OnInstallUpdate, OnRevealHome       func()
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
	anchor := ui.Column(c.Key("about-heading")).FillWidth().MaxWidth(560).MinWidth(0).Gap(12)
	storage := ui.Local(anchor, "metadata", func() credentials.StorageStatus {
		if a.ReadCredentialStorage != nil {
			return a.ReadCredentialStorage()
		}
		return credentials.StorageStatus{Mode: "unknown"}
	})
	linkError := ui.Local(anchor, "error", func() string { return "" })
	anchor.Children(func() {
		ui.Text(c, i18n.T(i18n.KeySettingsAboutTitle)).FillWidth().FontSize(fonttheme.InterfaceSize(18)).FontWeight(600).TextColor(fg)
		ui.Text(c, providerCopy("本桌面应用与官方 Harness。", "This desktop app and the official Harness.")).FillWidth().FontSize(fonttheme.InterfaceSize(13)).TextColor(muted)
		if a.Loading {
			ui.Text(c, providerCopy("正在读取应用信息…", "Loading application information…")).FillWidth().TextColor(muted)
		}
		if a.Err != "" {
			ui.Text(c, a.Err).FillWidth().TextColor(fg)
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
			ui.Text(c, emptyDash(a.AppName)).FillWidth().FontSize(fonttheme.InterfaceSize(15)).Bold().TextColor(fg)
			ui.Text(c, providerCopy("桌面版本 ", "Desktop version ")+emptyDash(a.Version)).FillWidth().SingleLine().Tooltip(a.Version).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Text(c, "Based on DeepSeek Harness").FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				if a.OnCheckUpdates != nil && ui.Button(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates)).Disabled(a.UpdateBusy).Clicked() {
					a.OnCheckUpdates()
				}
				if a.OnInstallUpdate != nil && ui.Button(c, providerCopy("在线安装", "Install update")).Disabled(a.UpdateBusy).Clicked() {
					a.OnInstallUpdate()
				}
				if a.OnCopyVersion != nil && ui.Button(c, i18n.T(i18n.KeySettingsAboutCopyVersion)).Clicked() {
					a.OnCopyVersion()
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
				ui.Text(c, providerCopy("运行目录 ", "Runtime directory ")+a.HomePath).FillWidth().SingleLine().Tooltip(a.HomePath).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
			if a.OnRevealHome != nil && ui.Button(c, providerCopy("打开运行目录", "Open runtime directory")).Disabled(a.HomeBusy).Clicked() {
				a.OnRevealHome()
			}
		})
		for _, link := range aboutLinks() {
			button := ui.Button(c, link.Label).FillWidth().MinWidth(0).Height(32).BorderWidth(0).Background(ui.Transparent).TextColor(fg).Tooltip(link.URL).Disabled(a.OnOpenLink == nil)
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
		{providerCopy("发行版本", "Releases"), "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases"},
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
