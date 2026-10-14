package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// AboutAccess is the dedicated About settings page.
type AboutAccess struct {
	AppName          string
	Version          string
	Identifier       string
	UserData         string
	VendorRoot       string
	OnRevealUserData func()
	OnRevealLogs     func()
	OnRevealVendor   func()
	OnCheckUpdates   func()
	OnCopyUserData   func()
	OnCopyVersion    func()
	AnalyticsEnabled *bool
	OnRefreshAnalytics func()
}

func aboutPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.About == nil {
		return
	}
	a := m.About
	ui.Text(c, i18n.T(i18n.KeySettingsAboutTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsAboutHint)).FontSize(12).TextColor(muted)
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
			ui.Text(c, row.label).FontSize(12).TextColor(muted).Width(140)
			ui.Text(c, row.value).FontSize(12).TextColor(fg).Grow(1)
		})
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
		if a.OnRefreshAnalytics != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).FontSize(11).Clicked() {
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
