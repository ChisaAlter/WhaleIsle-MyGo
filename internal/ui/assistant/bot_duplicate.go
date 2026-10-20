package assistant

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botDuplicateAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIConfirmThisSourceAndCopyEffects)
}
func botDuplicateExecutionAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIConfirmCopiedGoalExecutionEffects)
}
func botDuplicateConfirmLabel() string {
	return i18n.T(i18n.KeyMiscAssistantConfirmDuplicateBot)
}

type BotDuplicateModel struct {
	State                                                         dshbotstore.BotDuplicateSnapshot
	HostLabel, SourceID                                           string
	OnRefresh, OnStart, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange                                                      func(field, value string)
	OnAcknowledge, OnAcknowledgeExecution                         func(bool)
	OnOpen                                                        func(id string)
}

func botDuplicateBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.BotDuplicate.OnRefresh == nil {
		return
	}
	botDuplicateView(c, m.BotDuplicate, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func botDuplicateConfigText(item pluginrpc.DshbotBotItem) string {
	config := map[string]json.RawMessage{}
	for _, key := range pluginrpc.DshbotBotDuplicateConfigKeys {
		if raw, ok := item.Profile[key]; ok {
			config[key] = raw
		}
	}
	raw, _ := json.MarshalIndent(config, "", "  ")
	return string(raw)
}
func botDuplicateReviewText(host string, cmd pluginrpc.DshbotBotDuplicateCommand) string {
	var source pluginrpc.DshbotBotItem
	for _, item := range cmd.Expected.Items {
		if item.ID == cmd.SourceID {
			source = item
			break
		}
	}
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantBotDuplicateConfig), host, cmd.Expected.Revision, source.Name, cmd.SourceID, source.SessionID, cmd.Name, cmd.ScratchCwd, botDuplicateConfigText(source))
}
func botDuplicateView(c *ui.Context, m BotDuplicateModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("native-bot-duplicate")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantDuplicateSelectedPersonalBot)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "复制原版支持的完整配置，新建资料和专属会话；副本不会自动打开。先读取当前来源，再填写明确的新名称。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReadDuplicateSource)).Disabled(blocked || s.Scope == "" || m.SourceID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "Host warning: "+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadTheFullCurrentSourceAndWriteA)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable5)).TextColor(accent)
		}
		if s.Created.ID != "" {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantReceiptLinkedCopy), s.Created.Name, s.Created.ID, s.Created.SessionID, botDuplicateConfigText(s.Created))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, "资料和会话回执已确认；记忆是否复制完整须看 Host warning，不能用资料回执替代记忆证明。单独打开可能创建代理、恢复会话并执行上方目标及工具/消息，可能产生费用。").FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			if m.OnOpen != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantOpenReceiptLinkedCopy)).Disabled(blocked).Clicked() {
				m.OnOpen(s.Created.ID)
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantPrepareDuplicate)).Disabled(blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		name := d.Name
		ui.TextInput(c.Key("bot-duplicate-name"), &name).Label(i18n.T(i18n.KeyMiscAssistantNewStableName120)).FillWidth().Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("name", name)
			}
		})
		cwd := d.ScratchCwd
		ui.TextInput(c.Key("bot-duplicate-cwd"), &cwd).Label(i18n.T(i18n.KeyMiscAssistantScratchCwdOverridden)).FillWidth().Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("scratchCwd", cwd)
			}
		})
		if s.Uncertain {
			ui.Text(c, "复制结果保持未知，不能重发。Host 自行生成新身份：同配置的新记录也不能证明来自本次请求。候选仅供人工核对，打开候选不会解除原 attempt 的未知状态；记忆复制状态同样未知。保护只保留在本应用进程。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if s.Pending != nil {
				ui.Text(c, botDuplicateReviewText(m.HostLabel, *s.Pending)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
			}
			if len(s.Candidates) == 0 {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingCandidateDoesNotEstablish)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
			for _, row := range s.Candidates {
				row := row
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantUnattributedCandidate), row.Name, row.ID, row.SessionID, botDuplicateConfigText(row))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "请先核对这个目录 Bot 的全部配置与目标。明确打开可能恢复会话、执行目标、调用工具并产生费用或消息；原复制结果仍保持未知。").FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
				if m.OnOpen != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantOpenInspectedCandidate)+row.ID).Disabled(blocked).Clicked() {
					m.OnOpen(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardDuplicateDraft)).Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewDuplicate)).Disabled(disabled || !s.Loaded || !s.Writable).Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			ui.Column(c.Key("bot-duplicate-confirmation")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewDuplicate2)).Bold().TextColor(fg)
				ui.Text(c, botDuplicateReviewText(m.HostLabel, r.Command)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "Host 保留上方配置，包括完整 objective、模型、能力、peer 来源、分组、memoryReview 和 notify；空模型配置仍使用 Host 当时的应用默认模型。非空 workspace 优先于 scratch cwd。Host 复制执行时读取的 bot 记忆；不复制 user 记忆或会话历史，记忆复制失败可能仅 warning。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, "新 ID、session ID、名称和时间戳；pinned/hidden=false、pinOrder=0、收件箱空、readInitialized=true、lastSeenSeq=0，成员空、maxRounds=3/maxSpeaks=10，运行/hold 状态和例行任务不复制。原 Bot 保留。本次不额外发送 prompt，不自动打开副本；未知结果锁定且不重试。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botDuplicateAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				hasGoal := pluginrpc.DshbotBotDuplicateHasObjective(r.Command)
				if hasGoal {
					ui.Text(c, "副本保留上方完整目标与轮数上限。Host 创建新会话/代理、首次 idle 或后续单独打开可能同步并执行目标、使用工具或发送消息，可能产生费用。不能承诺复制品处于暂停状态。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
					executionAck := r.ExecutionAcknowledged
					if ui.Checkbox(c, &executionAck, botDuplicateExecutionAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledgeExecution != nil {
						m.OnAcknowledgeExecution(executionAck)
					}
				}
				if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToDuplicateDraft)).Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, botDuplicateConfirmLabel()).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged || (hasGoal && !r.ExecutionAcknowledged)).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
