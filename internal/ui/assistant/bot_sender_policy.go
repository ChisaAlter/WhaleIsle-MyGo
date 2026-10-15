package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"slices"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

const botSenderPolicyAcknowledgement = "我理解来源规则和目标可能执行的影响 / I understand source-policy and execution effects"
const botSenderPolicyExpansionAcknowledgement = "我明确允许下方列出的消息/任务权限扩大 / I explicitly approve the access expansion shown below"

type BotSenderPolicyModel struct {
	State                                                         dshbotstore.SenderPolicySnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange                                                      func(pluginrpc.DshbotSenderPolicyValues)
	OnAcknowledge, OnExpansionAcknowledge                         func(bool)
}

func botSenderNames(ids []string, s pluginrpc.DshbotSenderPolicyStatus) string {
	if len(ids) == 0 {
		return "无 / None"
	}
	rows := []string{}
	for _, id := range ids {
		label := id
		for _, p := range s.Peers {
			if p.ID == id {
				name := p.Title
				if name == "" {
					name = p.Name
				}
				label = fmt.Sprintf("%s [id=%s]", name, id)
				if p.Hidden {
					label += " (已隐藏 / hidden)"
				}
				break
			}
		}
		rows = append(rows, label)
	}
	return strings.Join(rows, "\n")
}
func botSenderPolicySummary(s pluginrpc.DshbotSenderPolicyStatus, v pluginrpc.DshbotSenderPolicyValues) string {
	messages, tasks := pluginrpc.DshbotSenderPolicyAccess(s, v)
	rule := "指定 Bot / Selected Bots"
	if !v.Selected {
		rule = "默认空名单 / Default empty list: 普通消息允许同 catalog 当前及未来所有其他个人 Bot；直接任务只允许同群 Bot。"
	}
	return fmt.Sprintf("%s\n完整 allowedSenderIds: %q\n普通消息 / Ordinary messages:\n%s\n直接任务委派 / Direct task delegation:\n%s", rule, v.IDs, botSenderNames(messages, s), botSenderNames(tasks, s))
}
func botSenderPolicyBlock(c *ui.Context, m BotSenderPolicyModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil
	ui.Column(c.Key("bot-sender-policy-editor")).Gap(7).Children(func() {
		ui.Text(c, "消息与任务来源 / Message and task sources").FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "选择当前 Host 同一 catalog 的个人 Bot。默认空名单不是拒绝所有，也不允许所有任务。群组按现有成员规则；隐藏 Bot 仍是有效身份。").FontSize(11).TextColor(muted)
		if ui.Button(c, "刷新消息与任务来源 / Refresh source policy").Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted).Selectable()
		}
		for _, w := range s.Warnings {
			ui.Text(c, "Host warning: "+w).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读回核实；未知结果保留并锁定草稿，不自动重试 / Read back before saving").TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		before, err := pluginrpc.DshbotSenderPolicyValue(s.Status)
		if err != nil {
			ui.Text(c, "字段不完整，请使用现有 Bot Web / Incomplete profile").TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s · revision %d\n当前规则 / Current rule:\n%s", m.HostLabel, s.Status.Bot.Name, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botSenderPolicySummary(s.Status, before))).FontSize(12).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := pluginrpc.CloneDshbotSenderPolicyValues(s.Draft)
		selected := "默认规则 / Default rule"
		if draft.Selected {
			selected = "指定 Bot / Selected Bots"
		}
		ui.Select(c.Key("bot-sender-policy-mode"), &selected, []string{"默认规则 / Default rule", "指定 Bot / Selected Bots"}).FillWidth().Disabled(!editable).OnChange(func() {
			draft.Selected = selected == "指定 Bot / Selected Bots"
			draft.IDs = []string{}
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		if draft.Selected {
			ui.Text(c, "指定名单必须至少选择一个 Bot；只有所选者可发普通消息和直接任务，同群也不例外。取消最后一个不会自动变成默认规则。").FontSize(11).TextColor(muted)
			for _, peer := range s.Status.Peers {
				peer := peer
				checked := slices.Contains(draft.IDs, peer.ID)
				label := botSenderNames([]string{peer.ID}, s.Status)
				if ui.Checkbox(c.Key("sender-peer-"+peer.ID), &checked, label).Disabled(!editable).Changed() && m.OnChange != nil {
					next := pluginrpc.CloneDshbotSenderPolicyValues(draft)
					if checked {
						next.IDs = append(next.IDs, peer.ID)
					} else {
						next.IDs = slices.DeleteFunc(next.IDs, func(id string) bool { return id == peer.ID })
					}
					m.OnChange(next)
				}
			}
		}
		ui.Text(c, "草稿完整规则 / Draft rule:\n"+botSenderPolicySummary(s.Status, draft)).FontSize(12).TextColor(fg).Selectable()
		if ui.Button(c, "丢弃来源草稿 / Discard source draft").Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅消息与任务来源 / Review source policy").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, "采用匹配的来源读回 / Adopt source readback").Disabled(blocked).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		cmd := r.Command
		old, _ := pluginrpc.DshbotSenderPolicyValue(cmd.Expected)
		expands := pluginrpc.DshbotSenderPolicyExpands(cmd)
		ui.Column(c.Key("bot-sender-policy-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, "确认消息与任务来源 / Confirm source policy").Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s · revision %d\n修改前 / Before:\n%s\n修改后 / After:\n%s", m.HostLabel, cmd.Expected.Bot.ID, cmd.Expected.Bot.SessionID, cmd.Expected.Revision, botSenderPolicySummary(cmd.Expected, old), botSenderPolicySummary(cmd.Expected, cmd.Values))).FontSize(12).TextColor(fg).Selectable()
			if expands {
				ui.Text(c, "权限扩大：新增允许者见修改后完整名单。切换默认规则还允许未来加入同 catalog 的个人 Bot 发普通消息；同群关系今后改变会影响默认任务权限。指定 Bot 可直接委派任务，即使不在同群。").FontSize(12).Bold().TextColor(accent)
			}
			ui.Text(c, "更改不撤销已发送/入队消息，不取消已创建或在途任务、模型和工具。现有失败/已取消任务重试会重新检查委派权限。此处没有拒绝全部的空名单语义，也没有外部 IM 身份或通配字符串授权。").FontSize(12).TextColor(accent)
			ui.Text(c, "bot/update 会校验保留的模型和能力、写审计、更新会话展示并尝试重命名，还会 requestSync：live idle 会话现在或下次 idle 可能创建目标、重建已暂停目标或修改 active 目标，调用模型、执行工具、发送外部消息并产生费用。冷会话不会在此保存路径解析，之后变为 live/idle 时可能同步；同步失败可能仅记 Host 日志。").FontSize(12).TextColor(accent)
			ui.Text(c, "保留的目标 / Retained objective: "+string(cmd.Expected.Profile["objective"])+"\n保留的模型 / Retained model: "+string(cmd.Expected.Profile["model"])).FontSize(11).TextColor(muted).Selectable()
			ui.Text(c, "只替换 allowedSenderIds。完整资料和原 revision 精确提交，不刷新替代审阅版本；CAS 不原子锁定目标或在途执行。回执只证明资料，不证明同步/重命名/执行完成。关闭不撤回已发送请求；未知结果只读回，不自动重试。").FontSize(11).TextColor(muted)
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botSenderPolicyAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if expands {
				ack := r.ExpansionAcknowledged
				if ui.Checkbox(c, &ack, botSenderPolicyExpansionAcknowledgement).Disabled(blocked).Changed() && m.OnExpansionAcknowledge != nil {
					m.OnExpansionAcknowledge(ack)
				}
			}
			if ui.Button(c, "取消来源保存 / Cancel source save").Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, "确认保存来源规则 / Confirm source save").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged || (expands && !r.ExpansionAcknowledged)).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
