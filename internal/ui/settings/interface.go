package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"

	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
)

// InterfaceAccess is the dedicated Interface settings page (locale + chrome prefs).
type InterfaceAccess struct {
	Shortcuts   *shortcuts.Editor
	Locale      *string
	CloseToTray *bool
	OpenAtLogin *bool
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
	ui.Text(c, i18n.T(i18n.KeySettingsInterfaceTitle)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsInterfaceHint)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
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
		ui.Checkbox(c, a.CloseToTray, "")
	}, fg, muted)
	desktopRow(c, i18n.T(i18n.KeySettingsDesktopOpenAtLogin), func() {
		if a.OpenAtLogin == nil {
			return
		}
		ui.Checkbox(c, a.OpenAtLogin, "")
	}, fg, muted)
	desktopRow(c, i18n.T(i18n.KeySettingsInterfaceAutoStart), func() {
		if a.AutoStartDesktop == nil {
			return
		}
		ui.Checkbox(c, a.AutoStartDesktop, "")
	}, fg, muted)
	ui.Text(c, i18n.T(i18n.KeySettingsInterfaceAutoStartDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	desktopRow(c, i18n.T(i18n.KeySettingsInterfaceViewTabs), func() {
		if a.ShowViewTabs == nil {
			return
		}
		ui.Checkbox(c, a.ShowViewTabs, "")
	}, fg, muted)
	ui.Text(c, i18n.T(i18n.KeySettingsInterfaceViewTabsDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	if a.ShowArchivedList != nil {
		desktopRow(c, providerCopy("显示归档列表", "Show archived list"), func() { ui.Checkbox(c, a.ShowArchivedList, "") }, fg, muted)
	}
	desktopRow(c, i18n.T(i18n.KeySettingsInterfaceAutoAllow), func() {
		if a.AutoAllowTools == nil {
			return
		}
		ui.Checkbox(c, a.AutoAllowTools, "")
	}, fg, muted)
	ui.Text(c, i18n.T(i18n.KeySettingsInterfaceAutoAllowDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	if a.OnSave != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
		a.OnSave()
	}
	shortcutsPane(c, a.Shortcuts, fg, muted)
}
