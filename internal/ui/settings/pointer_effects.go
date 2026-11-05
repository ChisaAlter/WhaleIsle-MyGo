package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
)

func pointerEffectsPane(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	if g == nil {
		return
	}
	anchor := ui.Column(c.Key("pointer-effects-settings")).FillWidth().MinWidth(0)
	open := ui.Local(anchor, "open", func() bool { return false })
	p := ui.Local(anchor, "draft", func() wallpapers.Preferences { return g.currentPreferences().Clone() })
	if settingsSearchReveal(c, "pointer-effects-settings") {
		*p = g.currentPreferences().Clone()
		*open = true
	}
	anchor.Children(func() {
		withRowGear(rowGear{label: providerCopy("指针特效设置", "Pointer effect settings"), disabled: g.Save == nil, onOpen: func() {
			*p = g.currentPreferences().Clone()
			*open = true
			g.Status = ""
		}}, func() {
			enabled := g.currentPreferences().CursorEnabled
			bodyLocalToggle(c, "cursor-enabled", providerCopy("指针特效", "Cursor effects"), providerCopy("移动指针即可预览；系统减少动态效果会关闭动画。", "Move the pointer to preview. Reduced motion disables the animation."), &enabled, g.Save == nil, fg, muted, func(value bool) {
				wallpaperChange(c, g, func(p *wallpapers.Preferences) { p.CursorEnabled = value })
			})
		})
	})
	if !*open {
		return
	}
	busy := g.pendingClose == open && g.Pending != nil && g.Pending()
	bodyDialogFooter(c, open, providerCopy("指针特效设置", "Pointer effect settings"), func() {
		g.pointerShown = true
		if g.pointerPreview == nil {
			g.pointerPreview = &ui.DecorativePreview{}
		}
		preview := p.Clone()
		preview.CursorEnabled = true
		config, _ := chrome.CursorEffectConfig(preview, c.Theme().Accent, c.Preferences().ReduceMotion)
		g.pointerPreview.Configure(config)
		ui.Box(c).Label(providerCopy("指针效果预览", "Pointer effect preview")).FillWidth().Height(180).Clip().Background(c.Theme().Surface).HandleInput(g.pointerPreview.Input).DrawOver(g.pointerPreview.Draw)
		bodyPreferenceRow(c, "cursor-effect", providerCopy("指针效果", "Pointer effect"), "", fg, muted, func() {
			appearanceChoiceCards(c, "cursor-kind-cards", &p.CursorEffect, [][2]string{{"trail", providerCopy("像素拖尾", "Pixel trail")}, {"splash", providerCopy("流体飞溅", "Fluid splash")}}, func(id string) []string {
				if id == "trail" {
					return []string{"#22d3ee", "#a78bfa", "#f472b6"}
				}
				return []string{"#60a5fa", "#2dd4bf", "#34d399"}
			}, g.Save == nil || busy)
		})
		bodyPreferenceRow(c, "cursor-preset", providerCopy("指针配色", "Pointer colors"), "", fg, muted, func() {
			if appearanceChoiceCards(c, "cursor-preset-cards", &p.CursorPreset, appearancePresetChoices(true), func(id string) []string {
				q := wallpapers.Defaults()
				q.ApplyCursorPreset(id)
				return q.CursorColors[:]
			}, g.Save == nil || busy) && p.CursorPreset != "custom" {
				p.ApplyCursorPreset(p.CursorPreset)
			}
		})
		beforeSpeed, beforeSize := p.CursorSpeed, p.CursorSize
		wallpaperSliderStep(c, "cursor-speed", providerCopy("指针速度", "Pointer speed"), &p.CursorSpeed, 20, 300, 10, g.Save == nil || busy, fg, muted)
		wallpaperSliderStep(c, "cursor-size", providerCopy("指针大小", "Pointer size"), &p.CursorSize, 25, 300, 5, g.Save == nil || busy, fg, muted)
		if beforeSpeed != p.CursorSpeed || beforeSize != p.CursorSize {
			p.CursorPreset = "custom"
		}
		for i := range p.CursorColors {
			bodyPreferenceRow(c, fmt.Sprintf("cursor-color-%d", i), fmt.Sprintf(providerCopy("颜色 %d", "Color %d"), i+1), providerCopy("留空跟随主题。", "Leave blank to follow the theme."), fg, muted, func() {
				if appearanceColorWell(c, &p.CursorColors[i], fmt.Sprintf("cursor-color-%d", i), g.Save == nil || busy, true) {
					p.CursorPreset = "custom"
				}
			})
		}
		if g.Status != "" {
			ui.Text(c, g.Status).FillWidth().TextColor(c.Theme().Danger)
		}
	}, func() {
		if ui.Button(c, providerCopy("恢复指针默认", "Reset pointer effects")).Disabled(g.Save == nil || busy).Clicked() {
			p.CursorEffect = "trail"
			p.ApplyCursorPreset("default")
		}
		if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(busy).Clicked() {
			*open = false
		}
		if ui.PrimaryButton(c, providerCopy("保存", "Save")).Disabled(g.Save == nil || busy).Clicked() {
			if !wallpaperColorsValid(c, p.CursorColors[:]) {
				return
			}
			if wallpaperChange(c, g, func(next *wallpapers.Preferences) {
				next.CursorEffect, next.CursorPreset, next.CursorColors = p.CursorEffect, p.CursorPreset, p.CursorColors
				next.CursorSpeed, next.CursorSize = p.CursorSpeed, p.CursorSize
				next.CursorEnabled = true
			}) {
				g.closeWhenSaved(open)
			}
		}
	}, ui.DialogOptions{DismissDisabled: busy})
}
