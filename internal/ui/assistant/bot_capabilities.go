package assistant

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"slices"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const botCapabilityAcknowledgement = "我理解能力范围及已有目标可能恢复执行的影响 / I understand capability and objective execution effects"

type BotCapabilityModel struct {
	State                                                dshbotstore.CapabilitySnapshot
	HostLabel                                            string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange                                             func(group, field, value string)
	OnAcknowledge                                        func(bool)
}

func botCapabilityProfileText(s pluginrpc.DshbotCapabilityStatus, key string) string {
	var value string
	if json.Unmarshal(s.Profile[key], &value) == nil {
		return value
	}
	return string(s.Profile[key])
}
func botCapabilityPolicyText(p pluginrpc.DshbotCapabilityPolicy) string {
	names := strings.Join(p.Names, ", ")
	if names == "" {
		names = "(none)"
	}
	return p.Mode + " · names: " + names
}
func botCapabilitiesBlock(c *ui.Context, m BotCapabilityModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && s.Status.Catalog.Active && s.Review == nil
	ui.Column(c.Key("bot-capabilities")).Gap(7).Children(func() {
		ui.Text(c, "Bot 能力范围 / Bot capabilities").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "编辑 Host 实际发现的 tools / skills / mcp 范围。管理安装、凭据和其他资料仍使用现有 Bot Web 入口。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, "刷新能力与资料 / Refresh capabilities").Disabled(blocked || s.Scope.BotID == "").Clicked() {
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
			ui.Text(c, "先刷新核实；未确认结果时草稿锁定，禁止再次保存 / Read back before saving").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s · revision %d", m.HostLabel, s.Status.Bot.Name, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 设置只读或写权限无法核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		if !s.Status.Catalog.Active {
			ui.Text(c, "能力服务未激活，无法编辑 / Capability service inactive").TextColor(accent)
		}
		if !s.Status.Catalog.SkillsAvailable {
			ui.Text(c, "Skills 服务不可用；不能保存非空选定 Skills / Skills service unavailable").TextColor(muted)
		}
		ui.Text(c, "all 包括本组当前及以后出现的能力；selected + 空列表拒绝整组。Skills 还需要允许普通工具 skill；祖先 Bot 和群成员限制仍会叠加。这里不会安装或试运行能力。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		for _, group := range pluginrpc.DshbotCapabilityGroups {
			group := group
			p := s.Draft[group]
			ui.Column(c.Key("capability-group-" + group)).Gap(4).Children(func() {
				ui.Text(c, group+": "+botCapabilityPolicyText(p)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Row(c).Gap(6).Children(func() {
					for _, mode := range []string{"all", "selected"} {
						mode := mode
						if ui.Button(c, group+": "+mode).Disabled(!editable || p.Mode == mode).Clicked() && m.OnChange != nil {
							m.OnChange(group, "mode", mode)
						}
					}
				})
				entries := append([]pluginrpc.DshbotCapabilityEntry{}, s.Status.Catalog.Entries(group)...)
				known := []string{}
				for _, e := range entries {
					known = append(known, e.Name)
				}
				for _, name := range p.Names {
					if !slices.Contains(known, name) {
						entries = append(entries, pluginrpc.DshbotCapabilityEntry{Name: name, Description: "已不在发现列表；选定模式须移除后保存 / Unavailable; remove before saving"})
					}
				}
				if p.Mode == "selected" {
					for _, e := range entries {
						e := e
						checked := slices.Contains(p.Names, e.Name)
						if ui.Checkbox(c.Key(group+"-"+e.Name), &checked, e.Name).Disabled(!editable).Changed() && m.OnChange != nil {
							m.OnChange(group, "toggle", e.Name)
						}
						if e.Description != "" {
							ui.Text(c, e.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						}
					}
				}
			})
		}
		if ui.Button(c, "丢弃能力草稿 / Discard capability draft").Disabled(blocked || !s.Loaded || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅能力变更 / Review capability changes").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Review != nil {
			r := s.Review
			before := r.Command.Expected
			ui.Column(c.Key("capability-review")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, "确认能力资料保存 / Review capability save").Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · Session: %s · revision %d\n保持工作区 / Workspace: %s\n已有完整目标 / Existing objective: %s\nMax rounds: %s", m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, botCapabilityProfileText(before, "workspaceId"), botCapabilityProfileText(before, "objective"), botCapabilityProfileText(before, "objectiveMaxRounds"))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				for _, group := range pluginrpc.DshbotCapabilityGroups {
					ui.Text(c, group+"\nBefore: "+botCapabilityPolicyText(before.Policies[group])+"\nAfter: "+botCapabilityPolicyText(r.Command.Desired[group])).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				ui.Text(c, "将以此 revision 原样交回当前完整资料，仅替换上述能力配置。Host 会记录审计并更新会话展示、尝试重命名；已有模型也会重新验证。all 是动态整组授权，未来新增能力也在范围内；selected 只允许列出的名称。未改资料不会由客户端补默认值。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Text(c, "重要：原版 bot/update 还会同步已有目录目标，可能重建已暂停目标；已有 live 会话现在或下次 idle 时可能继续目标，产生模型调用、工具执行、外部消息及费用。不会解析冷会话立即运行，但后续变为 live 时可能同步。能力修改不撤销已经发生的操作。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, "未知回包先读回权威资料，草稿保持锁定，不自动重试。关闭或切换 Host/Bot 会清除此局部草稿和确认。 / Unknown outcomes require readback.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botCapabilityAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if ui.Button(c, "取消能力保存 / Cancel capability save").Disabled(blocked).Clicked() && m.OnDismiss != nil {
					m.OnDismiss()
				}
				if ui.Button(c, "确认保存能力 / Confirm capability save").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
					m.OnConfirm()
				}
			})
		}
	})
}
