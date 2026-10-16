package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	fonttheme "whaleisle/internal/ui/theme"
)

func petPane(c *ui.Context, p *pet.Companion, fg, muted ui.Color) {
	ui.Text(c, "Desktop companion / 桌面宠物").FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, "Independent sprite window. Default off. Drag the character to move it; the last deliberate drag position is restored on restart. / 独立图集宠物窗，默认关闭；拖动角色移动，重启恢复最后一次手动位置，自主移动不覆盖它。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, "Shows the selected task. Quick chat opens a compact card on Whale’s saved conversation; Main pet chat opens that same session in the main window. / 显示当前任务；聊聊打开紧凑聊天卡，宠物主会话打开相同持久会话；回到会话仍返回当前任务。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, "Feed file opens a native picker and changes only saved care stats; selected files stay untouched. / 喂文件只保存饱食/心情变化，不读取或改动所选文件。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, "Refresh bowl reads this app’s actual usage. Feed Token spends local game food only, never API credit. / 刷新粮碗读取本应用真实用量；喂 Token 只消费本地游戏粮点，不扣账户余额。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, "Pat saves the original mood/bond care effect; repeat pats use its ten-minute bond cooldown. / 摸摸头保存原版心情/亲密变化，重复互动沿用十分钟亲密冷却。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, "Sleep pauses rendering for this window; Wake uses the original mood −7, with no rest reward. / 睡觉临时暂停渲染；叫醒沿用原版心情 −7，无休息收益，重新开窗恢复清醒。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, "Optional THA4 neural animation uses verified local assets. Missing resources or runtime errors fall back to sprites. / 可选 THA4 神经渲染；资源或运行时不可用则回退图集。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, "Look asks before each screenshot and uses the pet-specific image route selected below; replies appear on the pet. / 看看每次截图前确认，使用下方独立选择的桌宠视觉路由；回复显示在桌宠上。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if p.Error != "" {
		ui.Text(c, p.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	ui.Checkbox(c, &p.Enabled, "Show companion / 显示桌宠")
	ui.Checkbox(c, &p.Neural, "THA4 animation (optional) / THA4 动画（可选）")
	ui.Text(c, fmt.Sprintf("大小 / Scale %.1f · 透明度 / Opacity %.0f%%", p.Behavior.Scale, p.Behavior.Opacity*100)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
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
	ui.Text(c, "Personality / 性格").FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	ui.Select(c, &p.Behavior.Personality, []string{"natural", "genki", "tsundere", "poison"}).FillWidth()
	ui.Text(c, "natural 自然 · genki 元气 · tsundere 傲娇 · poison 毒舌").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	ui.Text(c, "Activity / 活跃程度（quiet 安静 / balanced 均衡 / active 活泼）").FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	ui.Select(c, &p.Behavior.Activity, []string{"quiet", "balanced", "active"}).FillWidth()
	ui.Checkbox(c, &p.Behavior.Wander, "Wander around / 自主移动")
	ui.Checkbox(c, &p.Behavior.SelfTalk, "Idle self-talk / 自言自语")
	ui.Checkbox(c, &p.Behavior.PowerSave, "Power save after five idle minutes / 闲置五分钟后省电")
	ui.Checkbox(c, &p.Behavior.LockPosition, "Lock position / 锁定拖动")
	ui.Checkbox(c, &p.Behavior.ShiftToDrag, "Hold Shift to drag / 按住 Shift 拖动")
	ui.Checkbox(c, &p.Behavior.ClickSound, "Click sound / 点击音效")
	ui.Checkbox(c, &p.Behavior.ChatEnabled, "Enable pet chat / 允许桌宠聊天入口")
	ui.Text(c, "Save applies appearance and interaction settings without restarting the pet. / 保存后应用外观与交互设置，无需重建桌宠窗口。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
		ui.Text(c, "Saved skin is missing; the bundled image will be shown. / 已存皮肤缺失，暂用内置形象。").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	ui.Text(c, "Skin (empty = bundled Whale image) / 皮肤（空白为内置鲸鱼娘）").FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	ui.Select(c, &p.SkinID, options).FillWidth()
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "Refresh local skins / 刷新本地皮肤").Clicked() {
			p.Refresh()
		}
		if ui.Button(c, "Save / 保存").Clicked() {
			p.Save()
		}
	})
	if ui.Button(c, "恢复外观与行为默认 / Reset appearance and behavior").Disabled(p.Store == nil).Clicked() {
		confirmResetPetAppearanceBehavior(p)
	}
	ui.Text(c, "Reads validated Codex pet.json + WebP sprites from CODEX_HOME/pets. No package scripts, downloads or uploads. Works offline without Harness. / 仅读取本地安全皮肤；无需 Harness 连通。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
}
