package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const botObjectiveConfigAcknowledgement = "我理解保存可能立即启动或恢复目标执行 / I understand saving may start or resume execution"

type BotObjectiveConfigModel struct {
	State                                                dshbotstore.ObjectiveConfigSnapshot
	HostLabel                                            string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange                                             func(pluginrpc.DshbotObjectiveConfigDraft)
	OnAcknowledge                                        func(bool)
}

func botObjectiveConfigBlock(c *ui.Context, m BotObjectiveConfigModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && s.Review == nil
	ui.Column(c.Key("bot-objective-config")).Gap(7).Children(func() {
		ui.Text(c, "目标配置 / Objective configuration").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "为已有 Bot 新增或修改目录目标。目标按原版去除首尾空白后最多 2,000 UTF-16 单元；轮数上限为整数 1–200。保存可能立即执行，不是仅存草稿。清除仍使用上方清除目录目标入口。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "刷新目标配置 / Refresh objective configuration").Disabled(blocked || s.Scope.BotID == "").Clicked() {
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
		current, err := pluginrpc.DshbotObjectiveConfigValue(s.Status)
		if err != nil {
			ui.Text(c, "Host 未返回完整目标字段；不补默认值，请使用现有 Bot Web 入口").TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s · revision %d\n当前完整目录目标 / Current objective: %s\n当前轮数上限 / Current round cap: %d", m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, current.Objective, current.MaxRounds)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 设置只读或写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		ui.TextArea(c.Key("objective-config-text"), &draft.Objective).Label("目标 / Objective").FillWidth().Lines(3, 10).Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		ui.TextInput(c.Key("objective-config-rounds"), &draft.MaxRounds).Label("轮数上限 / Round cap (1–200)").FillWidth().Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		if ui.Button(c, "丢弃目标配置草稿 / Discard objective draft").Disabled(blocked || !s.Loaded || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅目标配置 / Review objective configuration").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		before := r.Command.Expected
		old, _ := pluginrpc.DshbotObjectiveConfigValue(before)
		ui.Column(c.Key("objective-config-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, "确认目标配置 / Review objective configuration").Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s · revision %d\n原完整目标 / Before objective:\n%s\n新完整目标（已去除首尾空白）/ After objective (trimmed):\n%s\n轮数上限 / Round cap: %d → %d", m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, old.Objective, r.Command.Desired.Objective, old.MaxRounds, r.Command.Desired.MaxRounds)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, fmt.Sprintf("沿用模型配置 / Configured model: %s\n沿用 workspaceId: %s\n沿用能力策略 / Capability policy: %s", botCapabilityProfileText(before, "model"), botCapabilityProfileText(before, "workspaceId"), botCapabilityProfileText(before, "capabilities"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ui.Text(c, "此 bot/update 保存会 requestSync：live idle 会话可能立即创建目标、重建已暂停或其他非 active 目标，或修改正在 active 的目标；运行中的会话在后续 idle 同步。可能调用模型、执行工具、发送外部消息并产生费用。未指定模型时沿用 Host 默认模型；轮数上限不等于费用或外部操作限额。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "保存路径不解析冷会话；冷会话后续变为 live/idle 时可能同步并执行。成功回执仅证明目录配置返回，不证明 live 目标同步成功或任务完成；原版同步失败可只记录 Host 日志而不返回警告，请另行刷新目标状态。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "只替换 objective 与 objectiveMaxRounds，其他完整资料原样交回；目录 revision CAS 不等于 live 目标原子锁。原版同时写审计、更新会话展示并尝试重命名。不会取消在途模型/工具，也不撤销已发生的外部操作。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, "需要独立确认执行影响后才发送保存；本页不另发 start/resume。未知回包先读回，保留并锁定草稿，不自动重试。关闭界面只清本地草稿，不撤回已发保存。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botObjectiveConfigAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, "取消目标配置保存 / Cancel objective configuration").Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, "确认保存目标配置 / Confirm objective configuration").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
