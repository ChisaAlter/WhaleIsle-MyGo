package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
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
	// RestartPolicy is the committed snapshot, separate from the editable draft.
	RestartPolicy     *config.RestartPreferences
	Fields            DesktopFields
	OnSave            func()
	OnDetectElectron  func()
	OnCheckUpdates    func()
	OnBrowseWorkspace func()
	LastAction        string
	Err               string
}

func desktopPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Desktop == nil {
		return
	}
	d := m.Desktop
	ui.Text(c, i18n.T(i18n.KeySettingsDesktopTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsDesktopHint)).FontSize(12).TextColor(muted)
	if d.Err != "" {
		ui.Text(c, d.Err).FontSize(12).TextColor(fg)
	}
	if d.LastAction != "" {
		ui.Text(c, d.LastAction).FontSize(11).TextColor(muted)
	}
	ui.Row(c).Gap(8).Children(func() {
		if d.OnDetectElectron != nil {
			if ui.Button(c, i18n.T(i18n.KeySettingsDesktopDetectElectron)).Clicked() {
				d.OnDetectElectron()
			}
		}
		if d.OnCheckUpdates != nil {
			if ui.Button(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates)).Clicked() {
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
				if ui.Button(c, i18n.T(i18n.KeySettingsDesktopBrowseWorkspace)).FontSize(11).Clicked() {
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
		ui.Text(c, label).FontSize(12).TextColor(fg)
		body()
	})
	_ = muted
}
