package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

func pointerEffectsPane(c *ui.Context, g *WallpaperEditor, fg, muted ui.Color) {
	p := &g.Prefs
	ui.Text(c, "指针特效与按钮光泽").Bold().TextColor(fg)
	ui.Checkbox(c, &p.CursorEnabled, "启用指针特效")
	ui.Select(c, &p.CursorEffect, []string{"trail", "splash"}).Label("指针效果")
	if ui.Select(c, &p.CursorPreset, []string{"default", "rainbow", "aurora", "sunset", "ocean", "sakura", "custom"}).Label("指针配色方案").Changed() && p.CursorPreset != "custom" {
		p.ApplyCursorPreset(p.CursorPreset)
	}
	ui.Text(c, fmt.Sprintf("速度 %.0f%%", p.CursorSpeed)).FontSize(fonttheme.InterfaceSize(11))
	if ui.StepSlider(c, &p.CursorSpeed, 20, 300, 1).Changed() {
		p.CursorPreset = "custom"
	}
	ui.Text(c, fmt.Sprintf("大小 %.0f%%", p.CursorSize)).FontSize(fonttheme.InterfaceSize(11))
	if ui.StepSlider(c, &p.CursorSize, 25, 300, 1).Changed() {
		p.CursorPreset = "custom"
	}
	for i := range p.CursorColors {
		if ui.TextInput(c, &p.CursorColors[i]).Placeholder("#rrggbb；留空跟随主题").Label(fmt.Sprintf("指针颜色 %d", i+1)).Changed() {
			p.CursorPreset = "custom"
		}
	}
	ui.Checkbox(c, &p.MetallicButtons, "启用全局按钮悬停金属光泽")
	ui.Text(c, "移动指针即可预览。trail 为渐隐像素轨迹；splash 使用原生 CPU 染料平流与扩散近似，不等同于原版 WebGL 流体。系统减少动态效果会关闭指针动画，并静止按钮光泽。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "保存指针与按钮效果").Clicked() {
			g.commit()
		}
		if ui.Button(c, "恢复指针默认").Clicked() {
			p.CursorEnabled = false
			p.CursorEffect = "trail"
			p.ApplyCursorPreset("default")
			p.MetallicButtons = false
		}
		if ui.Button(c, "撤销效果调整").Clicked() {
			g.Prefs = g.Saved.Clone()
		}
	})
}
