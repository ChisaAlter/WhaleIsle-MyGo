package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const botTextProfileAcknowledgement = "我理解寻址、人设、会话展示及目标可能执行的影响 / I understand address, persona, presentation and execution effects"

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
		return "(空 / empty)"
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
		ui.Text(c, "名称、显示标题与人设 / Name, display title and persona").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "name 是稳定寻址名称，必填，最多120个UTF-16单元。title 是显示标题，最多120，可清空以沿用名称。description 是描述/人设，最多12000，可清空。保存前按原版去除首尾空白，完整展示变更。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "刷新文本资料 / Refresh text profile").Disabled(blocked || s.Scope.BotID == "").Clicked() {
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
			ui.Text(c, "先读回核实；未知结果保留并锁定草稿，不自动重试 / Read back before saving").TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		current, err := pluginrpc.DshbotTextProfileValue(s.Status)
		if err != nil {
			ui.Text(c, "文本字段不完整，请使用现有 Bot Web 入口 / Incomplete profile; use Bot Web").TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot ID: %s · Session ID: %s · revision %d\n当前寻址名称 / Current name: %s\n当前显示标题 / Current title: %s\n当前完整描述/人设 / Current complete description:\n%s", m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, current.Name, botTextProfileOptional(current.Title), botTextProfileOptional(current.Description))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		ui.TextInput(c.Key("bot-text-name"), &draft.Name).Label("寻址名称 / Name (120)").FillWidth().Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		ui.TextInput(c.Key("bot-text-title"), &draft.Title).Label("显示标题 / Display title (120, optional)").FillWidth().Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		ui.TextArea(c.Key("bot-text-description"), &draft.Description).Label("完整描述/人设 / Complete description/persona (12000)").FillWidth().Lines(3, 10).Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		if ui.Button(c, "丢弃文本草稿 / Discard text draft").Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅文本资料 / Review text profile").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, "采用匹配文本读回 / Adopt matching text readback").Disabled(blocked).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		cmd := r.Command
		old, _ := pluginrpc.DshbotTextProfileValue(cmd.Expected)
		ui.Column(c.Key("bot-text-profile-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, "确认完整文本资料 / Confirm complete text profile").Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf("Host: %s\nBot ID: %s · Session ID: %s · revision %d\n原寻址名称 / Before name:\n%s\n新寻址名称（已trim）/ After name:\n%s\n原显示标题 / Before title:\n%s\n新显示标题（已trim）/ After title:\n%s\n最终显示名称 / Display name: %s → %s", m.HostLabel, cmd.Expected.Bot.ID, cmd.Expected.Bot.SessionID, cmd.Expected.Revision, old.Name, cmd.Values.Name, botTextProfileOptional(old.Title), botTextProfileOptional(cmd.Values.Title), botTextProfileDisplay(old), botTextProfileDisplay(cmd.Values))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, "原完整描述/人设 / Before complete description:\n"+botTextProfileOptional(old.Description)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, "新完整描述/人设（已trim）/ After complete description:\n"+botTextProfileOptional(cmd.Values.Description)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			if old.Name != cmd.Values.Name {
				ui.Text(c, "寻址名称 name 改变：旧的按名称或 @name 寻址可能不再匹配。Bot ID 与 canonical Session ID 保持不变；不会替你修改既有文本里的旧名称。名称唯一性由 Host 按原版大小写规则核定。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			}
			ui.Text(c, "title 优先用于显示名称，不参与路由、名称唯一性或 @mention 解析；清空 title 后显示 name。显示名称也进入 Bot 模型身份文本。description 会作为人设进入系统提示词，并影响群成员人设；修改可影响后续行为。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "此次 bot/update 保存会写审计、更新会话 presentation，并尝试 rename 会话标题。重命名或展示回滚可能失败，请核对 Host 警告。保存同时校验原模型/能力并 requestSync：live idle 会话可能现在或下次 idle 创建目标、重建已暂停目标或修改 active 目标，调用模型、执行工具、发送外部消息并产生费用。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "冷会话不在此保存路径为目标同步而解析，后续变为 live/idle 时可能执行；同步失败可能仅记 Host 日志。目录 revision CAS 不原子锁定 live 目标。完整资料回执不证明 presentation/rename 或目标同步成功，更不证明任务完成。不会取消在途执行或撤销已发生的操作。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, fmt.Sprintf("保留 workspaceId: %s\n保留 model: %s\n仅替换 name/title/description；其他完整资料原样保留。此页不主动 prompt/start/resume/open。关闭不撤回已发请求；未知结果只读回，不自动重试。", botCapabilityProfileText(cmd.Expected, "workspaceId"), botCapabilityProfileText(cmd.Expected, "model"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botTextProfileAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, "取消文本资料保存 / Cancel text profile save").Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, "确认保存文本资料 / Confirm text profile save").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
