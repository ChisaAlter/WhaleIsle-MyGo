package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// GlassThemeAccess is the appearance page's optional 壁纸与毛玻璃 theme: off by
// default. Off, the window paints flat theme surfaces and the wallpaper,
// ambient background and frosted-glass settings below are hidden, because
// none of that machinery runs.
type GlassThemeAccess struct {
	Enabled  *bool
	OnToggle func(bool) error
}

func glassThemeOn(g *GlassThemeAccess) bool {
	return g == nil || g.Enabled == nil || *g.Enabled
}

func glassThemeSwitchLabel(enabled bool) string {
	if enabled {
		return providerCopy("关闭壁纸与毛玻璃主题", "Turn off the wallpaper and glass theme")
	}
	return providerCopy("开启壁纸与毛玻璃主题", "Turn on the wallpaper and glass theme")
}

func glassThemePane(c *ui.Context, g *GlassThemeAccess, fg, muted ui.Color) {
	if g == nil || g.Enabled == nil {
		return
	}
	title := providerCopy("壁纸与毛玻璃主题", "Wallpaper and glass theme")
	block := ui.Column(c.Key("appearance-glass-theme")).FillWidth().MaxWidth(720).MinWidth(0).Gap(12)
	settingsSearchTarget(c, "appearance-glass-theme", title, block)
	failure := ui.Local(block, "error", func() settingsRefusal { return settingsRefusal{} })
	block.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
				ui.Text(c, providerCopy(
					"可选外观：窗口壁纸、动态背景和半透明毛玻璃面板。默认关闭，关闭时界面使用纯色主题表面，不解码壁纸，也不运行背景或玻璃效果。",
					"Optional look: a window wallpaper, the ambient background and translucent frosted panels. Off by default; when off the interface uses solid theme surfaces and decodes no wallpaper or runs no background or glass effect.")).
					FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
			})
			enabled := *g.Enabled
			if settingsPreferenceSwitch(c, &enabled, settingsSwitchInput).Label(glassThemeSwitchLabel(enabled)).Disabled(g.OnToggle == nil).Changed() && g.OnToggle != nil {
				if err := g.OnToggle(enabled); err != nil {
					failure.refuse(err)
				} else {
					*g.Enabled = enabled
					failure.refuse(nil)
				}
			}
		})
		settingsAlert(c, "appearance-glass-theme:error", failure.failed(), failure.sentence, failure.detail)
	})
}
