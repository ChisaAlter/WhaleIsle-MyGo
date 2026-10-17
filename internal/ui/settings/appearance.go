package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// AppearanceAccess is the dedicated Appearance settings page (theme + wallpaper).
type AppearanceAccess struct {
	Theme            *string
	WallpaperPath    *string
	WallpaperChoices []string
	OnBrowseWallpaper func()
	OnSave           func()
	LastAction       string
	Err              string
}

func appearancePane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Appearance == nil {
		return
	}
	a := m.Appearance
	ui.Text(c, i18n.T(i18n.KeySettingsAppearanceTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsAppearanceHint)).FontSize(12).TextColor(muted)
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(12).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(11).TextColor(muted)
	}
	desktopRow(c, i18n.T(i18n.KeySettingsDesktopTheme), func() {
		if a.Theme == nil {
			return
		}
		// Wire ids match config theme ids (Electron FAMILY_SEEDS / Resolve).
		opts := []string{"deepseek", "system", "deepseek.dark", "deepseek.light"}
		ui.Select(c, a.Theme, opts).FillWidth()
	}, fg, muted)
	desktopRow(c, i18n.T(i18n.KeySettingsDesktopWallpaper), func() {
		if a.WallpaperPath == nil {
			return
		}
		opts := append([]string{""}, a.WallpaperChoices...)
		ui.Select(c, a.WallpaperPath, opts)
		if len(a.WallpaperChoices) == 0 {
			ui.Text(c, i18n.T(i18n.KeySettingsAppearanceWallpaperDir)).FontSize(11).TextColor(muted)
		}
		ui.Row(c).Gap(8).Children(func() {
			if a.OnBrowseWallpaper != nil {
				if ui.Button(c, i18n.T(i18n.KeySettingsAppearanceBrowseWallpaper)).FontSize(11).Clicked() {
					a.OnBrowseWallpaper()
				}
			}
			if ui.Button(c, i18n.T(i18n.KeySettingsAppearanceClearWallpaper)).FontSize(11).Clicked() {
				*a.WallpaperPath = ""
			}
		})
	}, fg, muted)
	if a.OnSave != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
		a.OnSave()
	}
}
