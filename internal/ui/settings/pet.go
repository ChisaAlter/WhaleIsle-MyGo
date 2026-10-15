package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
)

func petPane(c *ui.Context, p *pet.Companion, fg, muted ui.Color) {
	ui.Text(c, "Desktop companion / 桌面宠物").FontSize(14).Bold().TextColor(fg)
	ui.Text(c, "Independent sprite window. Default off. Drag the character to move it; position is restored on restart. / 独立图集宠物窗，默认关闭；拖动角色移动，重启恢复位置。").FontSize(12).TextColor(muted)
	ui.Text(c, "Shows the selected task. Pet chat opens Whale’s own saved conversation without sending a message; Open chat returns to the task. / 显示当前任务；和宠物聊聊打开鲸鱼娘专属持久会话，不自动发送消息。回到会话仍返回当前任务。").FontSize(12).TextColor(muted)
	ui.Text(c, "Feed file opens a native picker and changes only saved care stats; selected files stay untouched. / 喂文件只保存饱食/心情变化，不读取或改动所选文件。").FontSize(12).TextColor(muted)
	ui.Text(c, "Refresh bowl reads this app’s actual usage. Feed Token spends local game food only, never API credit. / 刷新粮碗读取本应用真实用量；喂 Token 只消费本地游戏粮点，不扣账户余额。").FontSize(12).TextColor(muted)
	ui.Text(c, "Pat saves the original mood/bond care effect; repeat pats use its ten-minute bond cooldown. / 摸摸头保存原版心情/亲密变化，重复互动沿用十分钟亲密冷却。").FontSize(12).TextColor(muted)
	ui.Text(c, "Sleep pauses rendering for this window; Wake uses the original mood −7, with no rest reward. / 睡觉临时暂停渲染；叫醒沿用原版心情 −7，无休息收益，重新开窗恢复清醒。").FontSize(12).TextColor(muted)
	ui.Text(c, "Optional THA4 neural animation uses verified local assets. Missing resources or runtime errors fall back to sprites. / 可选 THA4 神经渲染；资源或运行时不可用则回退图集。").FontSize(12).TextColor(muted)
	ui.Text(c, "Look asks before each screenshot and uses the configured image-capable vision route in Settings → Models; replies appear on the pet. / 看看每次截图前确认，使用设置 → 模型中已配置的视觉路由；回复显示在桌宠上。").FontSize(12).TextColor(muted)
	if p.Error != "" {
		ui.Text(c, p.Error).FontSize(12).TextColor(fg)
	}
	ui.Checkbox(c, &p.Enabled, "Show companion / 显示桌宠")
	ui.Checkbox(c, &p.Neural, "THA4 animation (optional) / THA4 动画（可选）")
	ui.Text(c, fmt.Sprintf("大小 / Scale %.1f · 透明度 / Opacity %.0f%%", p.Behavior.Scale, p.Behavior.Opacity*100)).FontSize(12).TextColor(fg)
	ui.Row(c).Gap(6).Children(func() {
		if ui.Button(c, "− 大小").Clicked() {
			p.Behavior.Scale -= .1
		}
		if ui.Button(c, "+ 大小").Clicked() {
			p.Behavior.Scale += .1
		}
		if ui.Button(c, "− 透明度").Clicked() {
			p.Behavior.Opacity -= .05
		}
		if ui.Button(c, "+ 透明度").Clicked() {
			p.Behavior.Opacity += .05
		}
		p.Behavior = config.NormalizePetBehavior(p.Behavior)
	})
	ui.Checkbox(c, &p.Behavior.LockPosition, "Lock position / 锁定拖动")
	ui.Checkbox(c, &p.Behavior.ShiftToDrag, "Hold Shift to drag / 按住 Shift 拖动")
	ui.Checkbox(c, &p.Behavior.ClickSound, "Click sound / 点击音效")
	ui.Checkbox(c, &p.Behavior.ChatEnabled, "Enable pet chat / 允许桌宠聊天入口")
	ui.Text(c, "Save applies appearance and interaction settings without restarting the pet. / 保存后应用外观与交互设置，无需重建桌宠窗口。").FontSize(11).TextColor(muted)
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
