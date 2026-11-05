package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
)

func backgroundEffectPreference(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	anchor := ui.Column(c.Key("background-effect-settings")).FillWidth().MinWidth(0)
	open := ui.Local(anchor, "open", func() bool { return false })
	draft := ui.Local(anchor, "draft", func() wallpapers.Preferences { return g.currentPreferences().Clone() })
	if settingsSearchReveal(c, "background-effect-settings") {
		*draft = g.currentPreferences().Clone()
		*open = true
	}
	anchor.Children(func() {
		withRowGear(rowGear{label: providerCopy("背景特效设置", "Background effect settings"), disabled: g.Save == nil, onOpen: func() {
			*draft = g.currentPreferences().Clone()
			*open = true
			g.Status = ""
		}}, func() {
			enabled := g.currentPreferences().Effect == "gradient"
			hint := providerCopy("没有壁纸时显示缓慢流动的渐变光斑。", "Show slowly moving gradient glows when no wallpaper is set.")
			if enabled && g.hasEffectiveWallpaper(g.currentPreferences()) {
				hint = providerCopy("已启用；当前壁纸优先显示。", "Enabled; the current wallpaper takes precedence.")
			}
			bodyLocalToggle(c, "background-effect-mode", providerCopy("背景特效", "Background effect"), hint, &enabled, g.Save == nil, fg, muted, func(value bool) {
				wallpaperChange(c, g, func(p *wallpapers.Preferences) {
					p.Effect = "none"
					if value {
						p.Effect = "gradient"
					}
				})
			})
		})
	})
	if !*open {
		if g.backgroundPreview != nil {
			g.backgroundPreview.Close()
			g.backgroundPreview = nil
		}
		return
	}
	if g.backgroundPreview == nil {
		g.backgroundPreview = &chrome.BackgroundEffectPreview{}
	}
	busy := g.pendingClose == open && g.Pending != nil && g.Pending()
	bodyDialogFooter(c, open, providerCopy("背景特效设置", "Background effect settings"), func() {
		preview := draft.Clone()
		preview.Effect, preview.EffectExplicit = "gradient", true
		ui.Box(c).FillWidth().Height(160).Radius(12).Clip().Draw(g.backgroundPreview.Draw(preview, c.Preferences().ReduceMotion))
		bodyPreferenceRow(c, "background-preset", providerCopy("配色方案", "Color preset"), "", fg, muted, func() {
			if appearanceChoiceCards(c, "background-preset-cards", &draft.Preset, appearancePresetChoices(false), func(id string) []string { colors := wallpapers.PresetColors(id); return colors[:] }, g.Save == nil || busy) && draft.Preset != "custom" {
				draft.Colors = wallpapers.PresetColors(draft.Preset)
			}
		})
		bodyPreferenceRow(c, "background-variant", providerCopy("动画样式", "Animation style"), "", fg, muted, func() {
			if wallpaperChoiceSelect(c, &draft.Variant, providerCopy("动画样式", "Animation style"), [][2]string{{"orbs", providerCopy("光球", "Orbs")}, {"aurora", providerCopy("极光", "Aurora")}, {"chaos", providerCopy("混沌", "Chaos")}, {"rays", providerCopy("光束", "Rays")}}, g.Save == nil || busy) {
				draft.Preset = "custom"
			}
		})
		beforeSpeed, beforeCount := draft.Speed, draft.Count
		wallpaperSliderStep(c, "background-speed", providerCopy("速度", "Speed"), &draft.Speed, 20, 300, 10, g.Save == nil || busy, fg, muted)
		wallpaperSlider(c, "background-count", providerCopy("光团数量", "Orb count"), &draft.Count, 1, 5, g.Save == nil || busy, fg, muted)
		if draft.Speed != beforeSpeed || draft.Count != beforeCount {
			draft.Preset = "custom"
		}
		for i := range draft.Colors {
			bodyPreferenceRow(c, fmt.Sprintf("background-color-%d", i), fmt.Sprintf(providerCopy("颜色 %d", "Color %d"), i+1), "#RRGGBB", fg, muted, func() {
				if appearanceColorWell(c, &draft.Colors[i], fmt.Sprintf("background-color-%d", i), g.Save == nil || busy, true) {
					draft.Preset = "custom"
				}
			})
		}
		if g.Status != "" {
			ui.Text(c, g.Status).FillWidth().TextColor(c.Theme().Danger)
		}
	}, func() {
		if ui.Button(c, providerCopy("恢复默认", "Reset")).Disabled(g.Save == nil || busy).Clicked() {
			*draft = wallpapers.Defaults()
		}
		if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(busy).Clicked() {
			*open = false
		}
		if ui.PrimaryButton(c, providerCopy("保存", "Save")).Disabled(g.Save == nil || busy).Clicked() {
			if !wallpaperColorsValid(c, draft.Colors[:]) {
				return
			}
			if wallpaperChange(c, g, func(p *wallpapers.Preferences) {
				p.Preset, p.Colors, p.Speed, p.Count, p.Variant = draft.Preset, draft.Colors, draft.Speed, draft.Count, draft.Variant
			}) {
				g.closeWhenSaved(open)
			}
		}
	}, ui.DialogOptions{DismissDisabled: busy})
}
