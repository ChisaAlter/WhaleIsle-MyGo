package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pet"
)

func petPane(c *ui.Context, p *pet.Companion, fg, muted ui.Color) {
	ui.Text(c, "Desktop companion / 桌面宠物").FontSize(14).Bold().TextColor(fg)
	ui.Text(c, "Independent sprite window. Default off. Drag the character to move it; position is restored on restart. / 独立图集宠物窗，默认关闭；拖动角色移动，重启恢复位置。").FontSize(12).TextColor(muted)
	ui.Text(c, "Shows the selected task. Pet chat opens Whale’s own saved conversation without sending a message; Open chat returns to the task. / 显示当前任务；和宠物聊聊打开鲸鱼娘专属持久会话，不自动发送消息。回到会话仍返回当前任务。").FontSize(12).TextColor(muted)
	ui.Text(c, "Feed file opens a native picker and changes only saved care stats; selected files stay untouched. / 喂文件只保存饱食/心情变化，不读取或改动所选文件。").FontSize(12).TextColor(muted)
	ui.Text(c, "Optional THA4 neural animation uses verified local assets. Missing resources or runtime errors fall back to sprites. / 可选 THA4 神经渲染；资源或运行时不可用则回退图集。").FontSize(12).TextColor(muted)
	if p.Error != "" {
		ui.Text(c, p.Error).FontSize(12).TextColor(fg)
	}
	ui.Checkbox(c, &p.Enabled, "Show companion / 显示桌宠")
	ui.Checkbox(c, &p.Neural, "THA4 animation (optional) / THA4 动画（可选）")
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
