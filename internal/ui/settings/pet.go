package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pet"
)

func petPane(c *ui.Context, p *pet.Companion, fg, muted ui.Color) {
	ui.Text(c, "Desktop companion / 桌面宠物").FontSize(14).Bold().TextColor(fg)
	ui.Text(c, "Independent sprite window. Default off. Drag the character to move it; position is restored on restart. / 独立图集宠物窗，默认关闭；拖动角色移动，重启恢复位置。").FontSize(12).TextColor(muted)
	ui.Text(c, "Shows only the selected chat on the current connection. Idle does not mean success. Use Open chat to return; Hide closes this instance, Save shows it again. / 仅显示当前连接所选会话；未运行不代表成功。点击回会话；隐藏后保存可重新显示。").FontSize(12).TextColor(muted)
	ui.Text(c, "THA4 / Live2D is not connected. / 尚未接通 THA4 / Live2D 渲染。").FontSize(12).TextColor(muted)
	if p.Error != "" {
		ui.Text(c, p.Error).FontSize(12).TextColor(fg)
	}
	ui.Checkbox(c, &p.Enabled, "Show companion / 显示桌宠")
	options := []string{""}
	found := p.SkinID == ""
	for _, s := range p.Skins {
		options = append(options, s.ID)
		if s.ID == p.SkinID {
			found = true
		}
	}
	if !found {
		options = append(options, p.SkinID)
		ui.Text(c, "Saved skin is missing; the bundled image will be shown. / 已存皮肤缺失，暂用内置形象。").FontSize(12).TextColor(muted)
	}
	ui.Text(c, "Skin (empty = bundled Whale image) / 皮肤（空白为内置鲸鱼娘）").FontSize(12).TextColor(fg)
	ui.Select(c, &p.SkinID, options).FillWidth()
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "Refresh local skins / 刷新本地皮肤").Clicked() {
			p.Refresh()
		}
		if ui.Button(c, "Save / 保存").Clicked() {
			p.Save()
		}
	})
	ui.Text(c, "Reads validated Codex pet.json + WebP sprites from CODEX_HOME/pets. No package scripts, downloads or uploads. Works offline without Harness. / 仅读取本地安全皮肤；无需 Harness 连通。").FontSize(11).TextColor(muted)
}
