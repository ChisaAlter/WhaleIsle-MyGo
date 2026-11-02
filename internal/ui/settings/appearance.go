package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Artwork from ui-primitives/src/icons/index.tsx, used by ColorSchemeTiles.
var appearanceSchemeIcons = map[string]*ui.SVG{
	"deepseek.light": ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M8.00007 11.8117C10.1052 11.8117 11.8117 10.1052 11.8117 8.00007C11.8117 5.89499 10.1052 4.18848 8.00007 4.18848C5.89499 4.18848 4.18848 5.89499 4.18848 8.00007C4.18848 10.1052 5.89499 11.8117 8.00007 11.8117Z"/><path d="M13.3899 8H15.1499M11.8115 11.8115L13.0556 13.0556M8 13.3901V15.1501M4.18868 11.8115L2.94458 13.0556M2.6101 8H0.850098M4.18868 4.18856L2.94458 2.94446M8 2.6101V0.850098M11.8115 4.18856L13.0556 2.94446"/></svg>`)),
	"deepseek.dark":  ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M14.1127 8.70663C14.2576 8.60602 14.4627 8.71355 14.4386 8.88834C14.2901 9.96567 13.8731 10.9912 13.2229 11.8692C12.479 12.8735 11.4613 13.6421 10.2917 14.0829C9.1222 14.5236 7.85038 14.6179 6.62865 14.3543C5.40692 14.0907 4.28709 13.4805 3.40332 12.5967C2.51955 11.7129 1.90931 10.5931 1.64572 9.37135C1.38212 8.14962 1.47635 6.87779 1.91711 5.70825C2.35787 4.5387 3.12647 3.52103 4.13083 2.77714C5.00878 2.12689 6.03433 1.70994 7.11166 1.5614C7.28645 1.5373 7.39397 1.74238 7.29337 1.88734C6.68703 2.76099 6.37885 3.81241 6.42313 4.88345C6.47392 6.11194 6.98471 7.27645 7.85413 8.14587C8.72355 9.01529 9.88805 9.52608 11.1166 9.57687C12.1876 9.62114 13.239 9.31296 14.1127 8.70663Z"/></svg>`)),
	"system":         ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M13.5 2.5H2.5C1.94772 2.5 1.5 2.94772 1.5 3.5V11.5C1.5 12.0523 1.94772 12.5 2.5 12.5H13.5C14.0523 12.5 14.5 12.0523 14.5 11.5V3.5C14.5 2.94772 14.0523 2.5 13.5 2.5Z"/><path d="M5 14.5H11"/></svg>`)),
}

// AppearanceAccess is the dedicated Appearance settings page (theme + wallpaper).
type AppearanceAccess struct {
	Host              *BodySettingsAccess
	Background        *WallpaperEditor
	Preferences       *theme.Editor
	Theme             *string
	WallpaperPath     *string
	WallpaperChoices  []string
	OnBrowseWallpaper func()
	OnSetTheme        func(string) error
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
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, settingsActionText(a.LastAction)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	scheme := ui.Column(c.Key("appearance-scheme")).FillWidth().MaxWidth(720).MinWidth(0).Gap(12)
	schemeError := ui.Local(scheme, "error", func() string { return "" })
	scheme.Children(func() {
		ui.Text(c, providerCopy("颜色模式", "Color scheme")).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
		ui.Text(c, providerCopy("选择浅色、深色或跟随系统。", "Choose light, dark, or follow the system.")).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		if a.Theme != nil {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
				for _, mode := range []struct{ id, zh, en string }{{"deepseek.light", "浅色", "Light"}, {"deepseek.dark", "深色", "Dark"}, {"system", "跟随系统", "System"}} {
					selected := *a.Theme == mode.id || mode.id == "system" && *a.Theme == "deepseek" || mode.id == "deepseek.light" && *a.Theme == "light" || mode.id == "deepseek.dark" && *a.Theme == "dark"
					button := ui.Button(c, "").Label(providerCopy(mode.zh, mode.en)).Basis(180).Grow(1).MinWidth(0).Padding(20, 32).Radius(16).Border(.5, c.Theme().Border).Background(ui.Transparent).Selected(selected).Disabled(a.OnSetTheme == nil)
					if selected {
						// ColorSchemeTiles.themeCube.selected: module-platform fill
						// and the same .5 DIP border in neutral-bluish-400.
						button.Background(theme.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).SelectorFill)).Border(.5, ui.RGB(173, 178, 184))
					} else if button.Hovered() {
						button.Background(c.Theme().SurfaceHover)
					}
					button.Children(func() {
						ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, appearanceSchemeIcons[mode.id]).Size(16, 16).TextColor(fg)
							ui.Text(c, providerCopy(mode.zh, mode.en)).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
						})
					})
					if button.Clicked() && a.OnSetTheme != nil {
						if err := a.OnSetTheme(mode.id); err != nil {
							*schemeError = err.Error()
						} else {
							*a.Theme, *schemeError = mode.id, ""
						}
					}
				}
			})
		}
		if *schemeError != "" {
			ui.Text(c, *schemeError).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(fg)
		}
	})
	themeLibraryPane(c, a.Preferences, theme.Resolve(m.ThemeID).Active(m.SystemDark), fg, muted)
	wallpaperBackgroundPane(c, a.Background, fg, muted)
	themeTypographyPane(c, a.Preferences, fg, muted)
	// Advanced: conversation font size (moved from General), the typing effect and the
	// thinking glow (moved from Interface).
	bodyFontSizePreference(c, a.Preferences, fg, muted)
	bodyEffectPreference(c, a.Host, true, fg, muted)
	bodyEffectPreference(c, a.Host, false, fg, muted)
	if a.OnSave != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Clicked() {
		a.OnSave()
	}
}
