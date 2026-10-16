package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/credentials"
	fonttheme "whaleisle/internal/ui/theme"

	"whaleisle/internal/ui/i18n"
)

// AboutAccess is the dedicated About settings page.
type AboutAccess struct {
	ReadCredentialStorage func() credentials.StorageStatus
	OnOpenLink            func(string) error
	AppName               string
	Version               string
	Identifier            string
	UserData              string
	VendorRoot            string
	OnRevealUserData      func()
	OnRevealLogs          func()
	OnRevealVendor        func()
	OnCheckUpdates        func()
	OnCopyUserData        func()
	OnCopyVersion         func()
	AnalyticsEnabled      *bool
	OnRefreshAnalytics    func()
}

func aboutPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.About == nil {
		return
	}
	a := m.About
	heading := ui.Text(c.Key("about-heading"), i18n.T(i18n.KeySettingsAboutTitle)).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	storage := ui.Local(heading, "metadata", func() credentials.StorageStatus {
		if a.ReadCredentialStorage != nil {
			return a.ReadCredentialStorage()
		}
		return credentials.StorageStatus{Mode: "unknown"}
	})
	linkError := ui.Local(heading, "error", func() string { return "" })
	ui.Text(c, i18n.T(i18n.KeySettingsAboutHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	analytics := "—"
	if a.AnalyticsEnabled != nil {
		if *a.AnalyticsEnabled {
			analytics = i18n.T(i18n.KeySettingsAboutAnalyticsOn)
		} else {
			analytics = i18n.T(i18n.KeySettingsAboutAnalyticsOff)
		}
	}
	rows := []struct{ label, value string }{
		{i18n.T(i18n.KeySettingsAboutApp), emptyDash(a.AppName)},
		{i18n.T(i18n.KeySettingsAboutVersion), emptyDash(a.Version)},
		{i18n.T(i18n.KeySettingsAboutID), emptyDash(a.Identifier)},
		{i18n.T(i18n.KeySettingsAboutUserData), emptyDash(a.UserData)},
		{i18n.T(i18n.KeySettingsAboutVendor), emptyDash(a.VendorRoot)},
		{i18n.T(i18n.KeySettingsAboutAnalytics), analytics},
	}
	for _, row := range rows {
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, row.label).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Width(140)
			ui.Text(c, row.value).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
		})
	}

	ui.Text(c, providerCopy("凭据存储", "Credential storage")+": "+storage.Backend).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	ui.Text(c, credentialStorageDescription(*storage)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if storage.Error != "" {
		ui.Text(c, storage.Error).TextColor(fg)
	}
	if a.ReadCredentialStorage != nil && ui.Button(c, providerCopy("刷新凭据存储状态", "Refresh credential storage status")).Clicked() {
		*storage = a.ReadCredentialStorage()
	}
	for _, link := range aboutLinks() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, link.URL).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			if a.OnOpenLink != nil && ui.Button(c, link.Label).Clicked() {
				*linkError = ""
				if err := a.OnOpenLink(link.URL); err != nil {
					*linkError = err.Error()
				}
			}
		})
	}
	if *linkError != "" {
		ui.Text(c, *linkError).TextColor(fg)
	}
	ui.Row(c).Gap(8).Children(func() {
		if a.OnRevealUserData != nil && ui.Button(c, i18n.T(i18n.KeySettingsAboutRevealData)).Clicked() {
			a.OnRevealUserData()
		}
		if a.OnRevealLogs != nil && ui.Button(c, i18n.T(i18n.KeySettingsAboutRevealLogs)).Clicked() {
			a.OnRevealLogs()
		}
		if a.OnRevealVendor != nil && ui.Button(c, i18n.T(i18n.KeySettingsAboutRevealVendor)).Clicked() {
			a.OnRevealVendor()
		}
		if a.OnCopyUserData != nil && ui.Button(c, i18n.T(i18n.KeySettingsAboutCopyData)).Clicked() {
			a.OnCopyUserData()
		}
		if a.OnCopyVersion != nil && ui.Button(c, i18n.T(i18n.KeySettingsAboutCopyVersion)).Clicked() {
			a.OnCopyVersion()
		}
		if a.OnCheckUpdates != nil && ui.Button(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates)).Clicked() {
			a.OnCheckUpdates()
		}
		if a.OnRefreshAnalytics != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			a.OnRefreshAnalytics()
		}
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
