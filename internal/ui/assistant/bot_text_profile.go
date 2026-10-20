package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botTextProfileAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandAddressPersonaPresentatio)
}

type BotTextProfileModel struct {
	State                                                         dshbotstore.TextProfileSnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange                                                      func(pluginrpc.DshbotTextProfileValues)
	OnAcknowledge                                                 func(bool)
}

func botTextProfileDisplay(v pluginrpc.DshbotTextProfileValues) string {
	if v.Title != "" {
		return v.Title
	}
	return v.Name
}
func botTextProfileOptional(s string) string {
	if s == "" {
		return i18n.T(i18n.KeyMiscAssistantEmpty)
	}
	return s
}
func botTextProfileBlock(c *ui.Context, m BotTextProfileModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil
	ui.Column(c.Key("bot-text-profile")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantNameDisplayTitleAndPersona)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "name 是稳定寻址名称，必填，最多120个UTF-16单元。title 是显示标题，最多120，可清空以沿用名称。description 是描述/人设，最多12000，可清空。保存前按原版去除首尾空白，完整展示变更。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshTextProfile)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
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
		current, err := pluginrpc.DshbotTextProfileValue(s.Status)
		if err != nil {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantIncompleteProfileUseBotWeb2)).TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantProfileDetail), m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, current.Name, botTextProfileOptional(current.Title), botTextProfileOptional(current.Description))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		ui.TextInput(c.Key("bot-text-name"), &draft.Name).Label(i18n.T(i18n.KeyMiscAssistantName120)).FillWidth().Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		ui.TextInput(c.Key("bot-text-title"), &draft.Title).Label(i18n.T(i18n.KeyMiscAssistantDisplayTitle120Optional)).FillWidth().Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		ui.TextArea(c.Key("bot-text-description"), &draft.Description).Label(i18n.T(i18n.KeyMiscAssistantCompleteDescriptionPersona12000)).FillWidth().Lines(3, 10).Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardTextDraft)).Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewTextProfile)).Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptMatchingTextReadback)).Disabled(blocked).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		cmd := r.Command
		old, _ := pluginrpc.DshbotTextProfileValue(cmd.Expected)
		ui.Column(c.Key("bot-text-profile-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantConfirmCompleteTextProfile)).Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantProfileDiff), m.HostLabel, cmd.Expected.Bot.ID, cmd.Expected.Bot.SessionID, cmd.Expected.Revision, old.Name, cmd.Values.Name, botTextProfileOptional(old.Title), botTextProfileOptional(cmd.Values.Title), botTextProfileDisplay(old), botTextProfileDisplay(cmd.Values))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantBeforeDescription)+botTextProfileOptional(old.Description)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, "新完整描述/人设（已trim）/ After complete description:\n"+botTextProfileOptional(cmd.Values.Description)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			if old.Name != cmd.Values.Name {
				ui.Text(c, "寻址名称 name 改变：旧的按名称或 @name 寻址可能不再匹配。Bot ID 与 canonical Session ID 保持不变；不会替你修改既有文本里的旧名称。名称唯一性由 Host 按原版大小写规则核定。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			}
			ui.Text(c, "title 优先用于显示名称，不参与路由、名称唯一性或 @mention 解析；清空 title 后显示 name。显示名称也进入 Bot 模型身份文本。description 会作为人设进入系统提示词，并影响群成员人设；修改可影响后续行为。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "此次 bot/update 保存会写审计、更新会话 presentation，并尝试 rename 会话标题。重命名或展示回滚可能失败，请核对 Host 警告。保存同时校验原模型/能力并 requestSync：live idle 会话可能现在或下次 idle 创建目标、重建已暂停目标或修改 active 目标，调用模型、执行工具、发送外部消息并产生费用。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "冷会话不在此保存路径为目标同步而解析，后续变为 live/idle 时可能执行；同步失败可能仅记 Host 日志。目录 revision CAS 不原子锁定 live 目标。完整资料回执不证明 presentation/rename 或目标同步成功，更不证明任务完成。不会取消在途执行或撤销已发生的操作。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, fmt.Sprintf("保留 workspaceId: %s\n保留 model: %s\n仅替换 name/title/description；其他完整资料原样保留。此页不主动 prompt/start/resume/open。关闭不撤回已发请求；未知结果只读回，不自动重试。", botCapabilityProfileText(cmd.Expected, "workspaceId"), botCapabilityProfileText(cmd.Expected, "model"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botTextProfileAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelTextProfileSave)).Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmTextProfileSave)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
