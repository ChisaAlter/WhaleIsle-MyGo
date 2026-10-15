package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

type BotObjectiveModel struct {
	State                           dshbotstore.ObjectiveSnapshot
	HostLabel                       string
	OnRefresh, OnConfirm, OnDismiss func()
	OnReview                        func(string)
}

func botObjectiveActionLabel(action string) string {
	switch action {
	case "start":
		return "开始 / 恢复目标… / Start or resume objective…"
	case "pause":
		return "暂停目标推进… / Pause objective…"
	case "clear":
		return "清除目录目标… / Clear catalog objective…"
	}
	return action
}
func botObjectiveImpact(action string) string {
	switch action {
	case "start":
		return "会解析或恢复该 Bot 的冷会话、同步目录目标，并按 Host 当前目标状态创建、恢复或重新启用目标推进。随后可能调用模型、执行工具、发送外部消息并产生费用；不会因为收到回执就表示目标完成。"
	case "pause":
		return "仅在 Host 已有 active live 目标时请求暂停；冷会话或非 active 状态可能无变化返回。不清除目录目标，不等同于停止 Bot 的所有工作；不承诺中断正在运行的模型轮，已发生的工具或外部操作不会撤销。"
	case "clear":
		return "把此 Bot 的目录目标文本清空，并尝试清除其当前 live 目标。live 清除失败可作为 Host 警告返回；配置成功不等于 live 清除成功。这不是暂停或 Bot 全局停止，不承诺中断在途模型或撤销已发生的操作。"
	}
	return ""
}
func botObjectiveStatus(c *ui.Context, s pluginrpc.DshbotObjectiveStatus, fg, muted ui.Color) {
	if !s.GoalKnown {
		ui.Text(c, "该回执未包含 live 目标状态；请刷新核实 / Live goal not included; refresh").FontSize(11).TextColor(muted)
		return
	}
	if s.Goal == nil {
		ui.Text(c, "Host 本次未报告可见 live 目标；冷会话可能仍有目录目标，不能据此判定已停止 / No visible live goal in this snapshot").FontSize(11).TextColor(muted)
		return
	}
	g := s.Goal
	ui.Text(c, fmt.Sprintf("Live phase: %s · activation: %s\nRounds started: %d / %d", g.Phase, g.Activation, g.RoundsStarted, g.MaxGoalRounds)).FontSize(12).TextColor(fg).Selectable()
	if g.Blocker != nil {
		ui.Text(c, "Host blocker: "+g.Blocker.Code+" · "+g.Blocker.Message).FontSize(11).TextColor(muted).Selectable()
	}
}
func botObjectiveBlock(c *ui.Context, m BotObjectiveModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("bot-objective")).Gap(7).Children(func() {
		ui.Text(c, "Bot 目标循环 / Bot objective").FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "此处管理已配置的目录目标；新增或修改目标及轮数请使用现有 Bot Web 资料编辑器。普通会话目标面板不能替代 Bot 目录配置。 / Configure in the existing Bot Web profile editor.").FontSize(11).TextColor(muted)
		if ui.Button(c, "刷新 Bot 目标状态 / Refresh objective").Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(12).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted)
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "上次操作的 Host 警告 / Host warning: "+warning).FontSize(12).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "请刷新核实后再操作；旧快照或回执不代表实时状态 / Refresh before another action").FontSize(11).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		b := s.Status.Bot
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s · revision %d\n目录目标 / Catalog objective: %s\n目录轮数上限 / Catalog max rounds: %d", m.HostLabel, b.Name, b.ID, b.SessionID, s.Status.Revision, b.Objective, b.ObjectiveMaxRounds)).FontSize(12).TextColor(fg).Selectable()
		botObjectiveStatus(c, s.Status, fg, muted)
		if s.Status.ActivityError != "" {
			ui.Text(c, "Host 活动读取提示 / Activity warning: "+s.Status.ActivityError).FontSize(11).TextColor(muted).Selectable()
		}
		ui.Row(c).Gap(7).Children(func() {
			for _, action := range []string{"start", "pause", "clear"} {
				disabled := blocked || !s.Loaded || s.Review != nil || action == "start" && strings.TrimSpace(b.Objective) == ""
				if m.OnReview != nil && ui.Button(c, botObjectiveActionLabel(action)).Disabled(disabled).Clicked() {
					m.OnReview(action)
				}
			}
		})
		if s.Review != nil {
			r := s.Review
			before := r.Command.Expected
			ui.Column(c.Key("bot-objective-review")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, "确认 / Review: "+botObjectiveActionLabel(r.Command.Action)).FontSize(13).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s · revision %d\n完整目录目标 / Objective: %s\nMax rounds: %d", m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, before.Bot.Objective, before.Bot.ObjectiveMaxRounds)).FontSize(12).TextColor(fg).Selectable()
				botObjectiveStatus(c, before, fg, muted)
				ui.Text(c, botObjectiveImpact(r.Command.Action)).FontSize(12).TextColor(accent)
				if r.Command.Action == "clear" {
					ui.Text(c, "清除只对目录 revision 做 Host CAS；live 清除是随后尝试，须检查警告与刷新状态。").FontSize(11).TextColor(muted)
				} else {
					ui.Text(c, "原版 start/pause 没有原子 CAS 或幂等请求键。发送前会重读核对目录版本、配置及 live 摘要，检查后仍有状态变化窗口；不伪造 goal ID/revision。").FontSize(11).TextColor(muted)
				}
				ui.Text(c, "回包不明时先刷新查看权威状态；不自动重发，也不把刷新当作再次执行。 / Refresh uncertain outcomes; never auto-retry.").FontSize(11).TextColor(muted)
				ui.Row(c).Gap(8).Children(func() {
					if m.OnDismiss != nil && ui.Button(c, "取消目标操作 / Cancel objective action").Disabled(blocked).Clicked() {
						m.OnDismiss()
					}
					if m.OnConfirm != nil && ui.Button(c, "确认目标操作 / Confirm objective action").Disabled(blocked || !s.Loaded).Clicked() {
						m.OnConfirm()
					}
				})
			})
		}
	})
}
