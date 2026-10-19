package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// AppearanceAccess is the dedicated Appearance settings page (theme + wallpaper).
type AppearanceAccess struct {
	Host              *BodySettingsAccess
	Background        *WallpaperEditor
	Preferences       *theme.Editor
	Theme             *string
	WallpaperPath     *string
	WallpaperChoices  []string
	OnBrowseWallpaper func()
	OnSave            func()
	LastAction        string
	Err               string
}

func appearancePane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Appearance == nil {
		return
	}
	a := m.Appearance
	bodyHostState(c, a.Host, muted)
	bodyHeading(c, providerCopy("颜色模式", "Color scheme"), providerCopy("选择浅色、深色或跟随系统。", "Choose light, dark, or follow the system."), fg, muted)
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if a.Theme != nil {
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			for _, mode := range []struct{ id, zh, en string }{{"deepseek.light", "浅色", "Light"}, {"deepseek.dark", "深色", "Dark"}, {"system", "跟随系统", "System"}} {
				selected := *a.Theme == mode.id || mode.id == "system" && *a.Theme == "deepseek"
				if ui.Button(c, providerCopy(mode.zh, mode.en)).Width(120).Height(64).Selected(selected).Disabled(a.OnSave == nil).Clicked() {
					*a.Theme = mode.id
				}
			}
		})
	}
	themeLibraryPane(c, a.Preferences, fg, muted)
	desktopRow(c, i18n.T(i18n.KeySettingsDesktopWallpaper), func() {
		if a.WallpaperPath == nil {
			return
		}
		opts := append([]string{""}, a.WallpaperChoices...)
		ui.Select(c, a.WallpaperPath, opts)
		if len(a.WallpaperChoices) == 0 {
			ui.Text(c, i18n.T(i18n.KeySettingsAppearanceWallpaperDir)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Row(c).Gap(8).Children(func() {
			if a.OnBrowseWallpaper != nil {
				if ui.Button(c, i18n.T(i18n.KeySettingsAppearanceBrowseWallpaper)).FontSize(theme.InterfaceSize(11)).Clicked() {
					a.OnBrowseWallpaper()
				}
			}
			if ui.Button(c, i18n.T(i18n.KeySettingsAppearanceClearWallpaper)).FontSize(theme.InterfaceSize(11)).Clicked() {
				*a.WallpaperPath = ""
			}
		})
	}, fg, muted)
	wallpaperBackgroundPane(c, a.Background, fg, muted)
	themeTypographyPane(c, a.Preferences, fg, muted)
	bodyEffectPreference(c, a.Host, true, fg, muted)
	if a.OnSave != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
		a.OnSave()
	}
}
