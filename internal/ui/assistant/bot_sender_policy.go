package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"slices"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botSenderPolicyAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandSourcePolicyAndExecution)
}
func botSenderPolicyExpansionAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIExplicitlyApproveTheAccessExpansio)
}

type BotSenderPolicyModel struct {
	State                                                         dshbotstore.SenderPolicySnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange                                                      func(pluginrpc.DshbotSenderPolicyValues)
	OnAcknowledge, OnExpansionAcknowledge                         func(bool)
}

func botSenderNames(ids []string, s pluginrpc.DshbotSenderPolicyStatus) string {
	if len(ids) == 0 {
		return i18n.T(i18n.KeyMiscAssistantNone)
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
					label += i18n.T(i18n.KeyMiscAssistantHidden)
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
	rule := i18n.T(i18n.KeyMiscAssistantSelectedBots)
	if !v.Selected {
		rule = i18n.T(i18n.KeyMiscAssistantDefaultEmptyList)
	}
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantSenderPolicySummary), rule, v.IDs, botSenderNames(messages, s), botSenderNames(tasks, s))
}
func botSenderPolicyBlock(c *ui.Context, m BotSenderPolicyModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil
	ui.Column(c.Key("bot-sender-policy-editor")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantMessageAndTaskSources)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "选择当前 Host 同一 catalog 的个人 Bot。默认空名单不是拒绝所有，也不允许所有任务。群组按现有成员规则；隐藏 Bot 仍是有效身份。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshSourcePolicy)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		for _, w := range s.Warnings {
			ui.Text(c, "Host warning: "+w).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadBackBeforeSaving4)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		before, err := pluginrpc.DshbotSenderPolicyValue(s.Status)
		if err != nil {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantIncompleteProfile)).TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantCurrentRuleDetail), m.HostLabel, s.Status.Bot.Name, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botSenderPolicySummary(s.Status, before))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := pluginrpc.CloneDshbotSenderPolicyValues(s.Draft)
		selected := i18n.T(i18n.KeyMiscAssistantDefaultRule)
		if draft.Selected {
			selected = i18n.T(i18n.KeyMiscAssistantSelectedBots)
		}
		ui.Select(c.Key("bot-sender-policy-mode"), &selected, []string{i18n.T(i18n.KeyMiscAssistantDefaultRule), i18n.T(i18n.KeyMiscAssistantSelectedBots)}).FillWidth().Disabled(!editable).OnChange(func() {
			draft.Selected = selected == i18n.T(i18n.KeyMiscAssistantSelectedBots)
			draft.IDs = []string{}
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		if draft.Selected {
			ui.Text(c, "指定名单必须至少选择一个 Bot；只有所选者可发普通消息和直接任务，同群也不例外。取消最后一个不会自动变成默认规则。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantDraftRule)+botSenderPolicySummary(s.Status, draft)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardSourceDraft)).Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewSourcePolicy)).Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptSourceReadback)).Disabled(blocked).Clicked() && m.OnAdopt != nil {
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
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantConfirmSourcePolicy)).Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantSenderPolicyDiff), m.HostLabel, cmd.Expected.Bot.ID, cmd.Expected.Bot.SessionID, cmd.Expected.Revision, botSenderPolicySummary(cmd.Expected, old), botSenderPolicySummary(cmd.Expected, cmd.Values))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			if expands {
				ui.Text(c, "权限扩大：新增允许者见修改后完整名单。切换默认规则还允许未来加入同 catalog 的个人 Bot 发普通消息；同群关系今后改变会影响默认任务权限。指定 Bot 可直接委派任务，即使不在同群。").FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(accent)
			}
			ui.Text(c, "更改不撤销已发送/入队消息，不取消已创建或在途任务、模型和工具。现有失败/已取消任务重试会重新检查委派权限。此处没有拒绝全部的空名单语义，也没有外部 IM 身份或通配字符串授权。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "bot/update 会校验保留的模型和能力、写审计、更新会话展示并尝试重命名，还会 requestSync：live idle 会话现在或下次 idle 可能创建目标、重建已暂停目标或修改 active 目标，调用模型、执行工具、发送外部消息并产生费用。冷会话不会在此保存路径解析，之后变为 live/idle 时可能同步；同步失败可能仅记 Host 日志。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRetainedObjective)+string(cmd.Expected.Profile["objective"])+i18n.T(i18n.KeyMiscAssistantRetainedModel)+string(cmd.Expected.Profile["model"])).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ui.Text(c, "只替换 allowedSenderIds。完整资料和原 revision 精确提交，不刷新替代审阅版本；CAS 不原子锁定目标或在途执行。回执只证明资料，不证明同步/重命名/执行完成。关闭不撤回已发送请求；未知结果只读回，不自动重试。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botSenderPolicyAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if expands {
				ack := r.ExpansionAcknowledged
				if ui.Checkbox(c, &ack, botSenderPolicyExpansionAcknowledgement()).Disabled(blocked).Changed() && m.OnExpansionAcknowledge != nil {
					m.OnExpansionAcknowledge(ack)
				}
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelSourceSave)).Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmSourceSave)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged || (expands && !r.ExpansionAcknowledged)).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
