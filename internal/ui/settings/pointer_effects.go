package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func pointerEffectsPane(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	if g == nil {
		return
	}
	p := &g.Prefs
	bodyLocalToggle(c, "cursor-enabled", providerCopy("指针特效", "Cursor effects"), providerCopy("移动指针即可预览；系统减少动态效果会关闭动画。", "Move the pointer to preview. Reduced motion disables the animation."), &p.CursorEnabled, g.Save == nil, fg, muted, nil)
	bodyDisclosure(c, "pointer-effects-options", providerCopy("指针特效设置", "Pointer effect settings"), func() {
		bodyPreferenceRow(c, "cursor-effect", providerCopy("指针效果", "Pointer effect"), "", fg, muted, func() { ui.Select(c, &p.CursorEffect, []string{"trail", "splash"}).FillWidth().Disabled(g.Save == nil) })
		bodyPreferenceRow(c, "cursor-preset", providerCopy("指针配色", "Pointer colors"), "", fg, muted, func() {
			if ui.Select(c, &p.CursorPreset, []string{"default", "rainbow", "aurora", "sunset", "ocean", "sakura", "custom"}).FillWidth().Disabled(g.Save == nil).Changed() && p.CursorPreset != "custom" {
				p.ApplyCursorPreset(p.CursorPreset)
			}
		})
		wallpaperSlider(c, "cursor-speed", providerCopy("指针速度", "Pointer speed"), &p.CursorSpeed, 20, 300, g.Save == nil, fg, muted)
		wallpaperSlider(c, "cursor-size", providerCopy("指针大小", "Pointer size"), &p.CursorSize, 25, 300, g.Save == nil, fg, muted)
		for i := range p.CursorColors {
			bodyPreferenceRow(c, fmt.Sprintf("cursor-color-%d", i), fmt.Sprintf(providerCopy("颜色 %d", "Color %d"), i+1), providerCopy("留空跟随主题。", "Leave blank to follow the theme."), fg, muted, func() {
				if ui.TextInput(c, &p.CursorColors[i]).FillWidth().Disabled(g.Save == nil).Changed() {
					p.CursorPreset = "custom"
				}
			})
		}
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if ui.Button(c, providerCopy("保存指针效果", "Save pointer effects")).Disabled(g.Save == nil).Clicked() {
				g.commit()
			}
			if ui.Button(c, providerCopy("恢复指针默认", "Reset pointer effects")).Disabled(g.Save == nil).Clicked() {
				p.CursorEnabled = false
				p.CursorEffect = "trail"
				p.ApplyCursorPreset("default")
				p.MetallicButtons = false
			}
		})
		ui.Text(c, providerCopy("调整后保存以应用。", "Save to apply your changes.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	})
}
