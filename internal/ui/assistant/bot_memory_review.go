package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const botMemoryReviewAcknowledgement = "我理解模型读取、记忆改写及目标同步影响 / I understand model, memory and objective effects"

type BotMemoryReviewModel struct {
	State                                                dshbotstore.MemoryReviewSnapshot
	HostLabel                                            string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange, OnAcknowledge                              func(bool)
}

func botMemoryReviewValue(enabled bool) string {
	if enabled {
		return "开启 / enabled"
	}
	return "关闭 / disabled"
}
func botMemoryReviewBlock(c *ui.Context, m BotMemoryReviewModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && s.Review == nil
	ui.Column(c.Key("bot-memory-review")).Gap(7).Children(func() {
		ui.Text(c, "自主学习记忆 / Automatic memory learning").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "管理原版 memoryReview 配置。达到 Host 完成轮数阈值（默认 10，范围 1–100）或开始压缩时尝试后台检查；按事件触发，不创建例行任务。本页不推断当前生效阈值或后台健康状态。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "刷新自动记忆配置 / Refresh memory learning").Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "上次 Host 警告 / Host warning: "+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读回核实；未知结果下草稿保持锁定 / Read back before saving again").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		current, err := pluginrpc.DshbotMemoryReviewEnabled(s.Status)
		if err != nil {
			ui.Text(c, "Host 未返回有效 memoryReview；不补默认值，请使用现有 Bot Web 入口").TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s · revision %d\nHost 配置: %s", m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botMemoryReviewValue(current))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 设置只读或写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		if ui.Checkbox(c, &draft, "允许自动整理记忆 / Allow automatic memory learning").Disabled(!editable).Changed() && m.OnChange != nil {
			m.OnChange(draft)
		}
		ui.Text(c, "开启后，侧路模型可读取近期用户/助手文本、近期工作摘要及 bot/user 两轨记忆，并增改删持久记忆，产生模型费用。新增或替换项带 [auto]；删除/替换并非只限自动条目。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, "关闭只阻止后续新检查：不取消在途模型请求或记忆写入，不清除已有记忆。重新开启不重置已累计轮数或连续 3 次失败后的进程内停用状态。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "丢弃自动记忆草稿 / Discard memory learning draft").Disabled(blocked || !s.Loaded || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅自动记忆配置 / Review memory learning").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Review != nil {
			r := s.Review
			before := r.Command.Expected
			old, _ := pluginrpc.DshbotMemoryReviewEnabled(before)
			ui.Column(c.Key("memory-review-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, "确认自动记忆配置 / Review memory learning").Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s · revision %d\nmemoryReview: %s → %s\n配置模型 / Configured model: %s\n已有完整目标 / Existing objective: %s\nMax rounds: %s", m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, botMemoryReviewValue(old), botMemoryReviewValue(r.Command.Enabled), botCapabilityProfileText(before, "model"), botCapabilityProfileText(before, "objective"), botCapabilityProfileText(before, "objectiveMaxRounds"))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				if r.Command.Enabled {
					ui.Text(c, "将允许后续事件触发侧路模型，向 Bot 所选模型提供近期对话、两轨记忆与近期工作，并自动增改删持久记忆；模型未选时使用 Host 默认。可能产生费用，配置开启不保证立即检查或后台运行正常。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				} else {
					ui.Text(c, "将阻止后续新自动检查；不取消在途请求或写入，不清除已有记忆，不重置后台状态。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				}
				ui.Text(c, "此保存仅改变 memoryReview，其他资料原样保留并使用目录 revision CAS。原版 bot/update 同时会审计、更新会话展示/尝试重命名并同步已有目标，可能重建已暂停目标；live 会话现在或下次 idle 可能继续执行模型、工具、外部消息并产生费用。关闭自动记忆也不能避免这项资料保存副作用。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, "本页不提供立即检查/重置，不直接调用模型、创建周期任务或读取两轨记忆内容；未知回包先读回，草稿保持锁定，不自动重试。关闭界面只清本地草稿，不撤回已发保存。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botMemoryReviewAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if ui.Button(c, "取消自动记忆保存 / Cancel memory learning").Disabled(blocked).Clicked() && m.OnDismiss != nil {
					m.OnDismiss()
				}
				if ui.Button(c, "确认保存自动记忆配置 / Confirm memory learning").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
					m.OnConfirm()
				}
			})
		}
	})
}
