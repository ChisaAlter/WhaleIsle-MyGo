package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const botNotifyToggleLabel = "允许原版 Bot 通知 / Allow original Bot notifications"
const botNotifyAcknowledgement = "我理解通知范围与完整资料保存的执行影响 / I understand notification scope and profile-save execution effects"

type BotNotifyModel struct {
	State                                                         dshbotstore.NotifySnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange, OnAcknowledge                                       func(bool)
}

func botNotifyValue(v bool) string {
	if v {
		return "notify=true · 允许 / enabled"
	}
	return "notify=false · 关闭 / disabled"
}
func botNotifyBlock(c *ui.Context, m BotNotifyModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("bot-notify")).Gap(7).Children(func() {
		ui.Text(c, "个人 Bot 通知配置 / Personal Bot notification setting").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "原版 Bot Web 消费 notify：窗口未聚焦且系统支持、权限允许时，通知新待决请求、目标转为受阻，以及非 silent 的例行任务 completed/failed/interrupted 终态。首次观测仅建基线，不补发历史事件。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, "此入口编辑共享配置；本页不申请系统通知权限、不测试或投递通知。原版 Web 可能在后续事件首次请求权限；系统支持、权限和通知消费者运行情况需另行核实。此配置不控制 IM、普通聊天消息、其他应用通知或任务执行。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "刷新通知配置 / Refresh notification setting").Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).TextColor(muted).Selectable()
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "Host warning: "+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读回核实；未知结果保留并锁定，只读回，不自动重试 / Read back before saving").TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		current, err := pluginrpc.DshbotNotifyEnabled(s.Status)
		if err != nil {
			ui.Text(c, "notify 字段无法核实，不猜默认值，请用 Bot Web / Boolean unavailable").TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot ID: %s · Session ID: %s · revision %d\n已保存 / Saved: %s", m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botNotifyValue(current))).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil
		draft := s.Draft
		if ui.Checkbox(c, &draft, botNotifyToggleLabel).Disabled(!editable).Changed() && m.OnChange != nil {
			m.OnChange(draft)
		}
		ui.Text(c, "草稿 / Draft: "+botNotifyValue(s.Draft)).TextColor(fg)
		if ui.Button(c, "丢弃通知草稿 / Discard notification draft").Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅通知配置 / Review notification setting").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, "采用匹配通知读回 / Adopt matching notification readback").Disabled(blocked).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		before, _ := pluginrpc.DshbotNotifyEnabled(r.Command.Expected)
		ui.Column(c.Key("bot-notify-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, "确认通知配置 / Confirm notification setting").Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf("Host: %s\nBot ID: %s · Session ID: %s · revision %d\n之前 / Before: %s\n之后 / After: %s", m.HostLabel, r.Command.Expected.Bot.ID, r.Command.Expected.Bot.SessionID, r.Command.Expected.Revision, botNotifyValue(before), botNotifyValue(r.Command.Enabled))).TextColor(fg).Selectable()
			if r.Command.Enabled {
				ui.Text(c, "允许后续符合条件的通知；不是立即通知或历史重放。配置回执不证明通知投递，也不证明操作系统权限或后台消费者正常。").TextColor(accent)
			} else {
				ui.Text(c, "关闭后续通知资格；不撤回已显示/已排队通知或已发权限请求，不停止目标、任务、模型或工具，不关闭 IM/其他原生通知。配置回执不证明通知投递已停止。").TextColor(accent)
			}
			ui.Text(c, "虽然只改 notify，完整 bot/update 仍会审计、更新会话 presentation、尝试 rename，并校验原模型/能力及 requestSync。live idle 可能创建目标、重建已暂停目标或修改 active 目标；可能调用模型、执行工具、发送外部消息并产生费用。关闭通知也不会避免这些保存副作用。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "忙碌/冷会话可能后续 idle 同步，失败可能只记 Host 日志；资料回执不证明展示、重命名、目标同步或执行完成。revision CAS 不原子锁定 live 目标，不撤回在途或已发生操作。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, fmt.Sprintf("完整保留 hidden、名称、头像、模型、工作区及其他资料。workspaceId: %s\nmodel: %s\nobjective: %s\n不主动 prompt/start/resume/open，不发送测试通知。关闭界面不撤回已发请求；未知结果锁定，只读回，不自动重试。", botCapabilityProfileText(r.Command.Expected, "workspaceId"), botCapabilityProfileText(r.Command.Expected, "model"), botCapabilityProfileText(r.Command.Expected, "objective"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botNotifyAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, "取消通知保存 / Cancel notification save").Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, "确认保存通知配置 / Confirm notification setting").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
