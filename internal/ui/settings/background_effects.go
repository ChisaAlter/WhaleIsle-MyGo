package settings

import (
	"fmt"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

func hydrateBackgroundDraft(p *wallpapers.Preferences) {
	p.Preset = wallpapers.NormalizeEffectPreset(p.Preset)
	p.Variant = wallpapers.NormalizeEffectVariant(p.Variant)
	p.Speed = wallpapers.ClampEffectSpeed(p.Speed)
	p.Count = wallpapers.ClampEffectCount(p.Count)
	if p.Colors == [7]string{} && p.Preset != "default" && p.Preset != "custom" {
		p.Colors = wallpapers.PresetColors(p.Preset)
	}
}

func backgroundPresetSwatches(id string, custom [7]string, dark bool) []string {
	colors := wallpapers.PresetColors(id)
	if id == "custom" {
		colors = custom
	}
	defaults := chrome.GradientDefaultColors(dark)
	for i := range colors {
		if colors[i] == "" {
			colors[i] = defaults[i]
		}
	}
	return colors[:]
}

func backgroundColorSlotLabel(index int) string {
	switch index {
	case 0:
		return providerCopy("底色起点", "Base start")
	case 1:
		return providerCopy("底色终点", "Base end")
	default:
		return fmt.Sprintf(providerCopy("光斑 %d", "Bloom %d"), index-1)
	}
}

func backgroundEffectPreference(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	anchor := ui.Column(c.Key("background-effect-settings")).FillWidth().MinWidth(0)
	open := ui.Local(anchor, "open", func() bool { return false })
	draft := ui.Local(anchor, "draft", func() wallpapers.Preferences {
		p := g.currentPreferences().Clone()
		hydrateBackgroundDraft(&p)
		return p
	})
	if settingsSearchReveal(c, "background-effect-settings") {
		*draft = g.currentPreferences().Clone()
		hydrateBackgroundDraft(draft)
		g.failure = ""
		*open = true
	}
	anchor.Children(func() {
		withRowGear(rowGear{label: providerCopy("背景特效设置", "Background effect settings"), disabled: g.Save == nil, onOpen: func() {
			*draft = g.currentPreferences().Clone()
			hydrateBackgroundDraft(draft)
			g.failure = ""
			*open = true
			g.Status = ""
		}}, func() {
			enabled := g.currentPreferences().Effect == "gradient"
			hint := providerCopy("没有壁纸时显示缓慢流动的渐变光斑。", "Show slowly moving gradient glows when no wallpaper is set.")
			if enabled && g.hasEffectiveWallpaper(g.currentPreferences()) {
				hint = providerCopy("已启用；当前壁纸优先显示。", "Enabled; the current wallpaper takes precedence.")
			}
			bodyLocalToggle(c, "background-effect-mode", providerCopy("背景特效", "Background effect"), minimalOverrideHint(hint), &enabled, g.Save == nil, fg, muted, func(value bool) {
				wallpaperChange(c, g, func(p *wallpapers.Preferences) {
					p.Effect = "none"
					if value {
						p.Effect = "gradient"
					}
				})
			})
			if g.LowPowerGPU {
				ui.Text(c, i18n.T(i18n.KeySettingsParityThemeEffectLowPowerGPU)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				if i18n.Locale() == i18n.LocaleEN {
					ui.Text(c, i18n.T(i18n.KeySettingsParityThemeEffectLowPowerGPUAlt)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
			}
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
			if appearanceChoiceCards(c, "background-preset-cards", &draft.Preset, appearancePresetChoices(false), func(id string) []string {
				return backgroundPresetSwatches(id, draft.Colors, c.Theme().Dark)
			}, g.Save == nil || busy) && draft.Preset != "custom" {
				draft.Colors = wallpapers.PresetColors(draft.Preset)
			}
		})
		bodyPreferenceRow(c, "background-variant", providerCopy("动画样式", "Animation style"), "", fg, muted, func() {
			if wallpaperChoiceSelect(c, &draft.Variant, providerCopy("动画样式", "Animation style"), [][2]string{{"orbs", providerCopy("光球", "Orbs")}, {"aurora", providerCopy("极光", "Aurora")}, {"chaos", providerCopy("混沌", "Chaos")}, {"rays", providerCopy("光束", "Rays")}}, g.Save == nil || busy) {
				draft.Preset = "custom"
			}
		})
		speedChanged := wallpaperSliderStepFormat(c, "background-speed", providerCopy("速度", "Speed"), minimalOverrideHint(""), "%", &draft.Speed, 20, 300, 10, g.Save == nil || busy, fg, muted)
		countChanged := wallpaperSlider(c, "background-count", providerCopy("光团数量", "Orb count"), minimalOverrideHint(""), &draft.Count, 1, 5, g.Save == nil || busy, fg, muted)
		if speedChanged || countChanged {
			draft.Preset = "custom"
		}
		for i := range draft.Colors {
			bodyPreferenceRow(c, fmt.Sprintf("background-color-%d", i), backgroundColorSlotLabel(i), "#RRGGBB", fg, muted, func() {
				if appearanceColorWell(c, &draft.Colors[i], fmt.Sprintf("background-color-%d", i), g.Save == nil || busy, true) {
					draft.Preset = "custom"
				}
			})
		}
		if g.failure != "" {
			settingsAlert(c, "appearance-dialog", true, providerCopy("没能保存外观设置，请重试。", "The appearance setting was not saved. Try again."), g.failure)
		}
	}, func() {
		if kGhost(c, providerCopy("恢复默认", "Reset"), g.Save == nil || busy).Clicked() {
			defaults := wallpapers.Defaults()
			draft.Preset, draft.Colors, draft.Speed, draft.Count, draft.Variant = defaults.Preset, defaults.Colors, defaults.Speed, defaults.Count, defaults.Variant
			g.failure = ""
		}
		if kButtonDisabled(c, providerCopy("取消", "Cancel"), busy).Clicked() {
			*open = false
		}
		if kPrimary(c, providerCopy("保存", "Save"), g.Save == nil || busy).Clicked() {
			draft.Preset = wallpapers.NormalizeEffectPreset(draft.Preset)
			draft.Variant = wallpapers.NormalizeEffectVariant(draft.Variant)
			draft.Speed = wallpapers.ClampEffectSpeed(draft.Speed)
			draft.Count = wallpapers.ClampEffectCount(draft.Count)
			if !wallpaperColorsValid(g, draft.Colors[:]) {
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
