package assistant

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const roomEditAcknowledgement = "我理解群资料及会话展示变更的影响 / I understand group profile and session effects"
const roomEditConfirmLabel = "确认保存群聊设置 / Confirm group update"

type RoomEditModel struct {
	State                                                                  dshbotstore.RoomEditSnapshot
	HostLabel                                                              string
	OnRefresh, OnStart, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange                                                               func(field, value string)
	OnMember                                                               func(id string)
	OnAcknowledge                                                          func(bool)
}

func roomEditBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.RoomEdit.State.Scope == "" || m.RoomEdit.OnRefresh == nil {
		return
	}
	roomEditView(c, m.RoomEdit, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func roomEditMembers(ids []string, catalog pluginrpc.DshbotRoomCatalog) string {
	lines := []string{}
	for _, id := range ids {
		name, session := "不可用 / unavailable", ""
		for _, row := range catalog.Items {
			if row.ID == id && row.Kind == "bot" {
				name, session = row.Name, row.SessionID
				break
			}
		}
		lines = append(lines, fmt.Sprintf("%s · Bot %s · Session %s", name, id, session))
	}
	return strings.Join(lines, "\n")
}
func roomEditReviewText(cmd pluginrpc.DshbotRoomEditCommand) string {
	before, _ := json.MarshalIndent(cmd.Expected.Profile, "", "  ")
	after, _ := json.MarshalIndent(pluginrpc.DshbotRoomEditProfile(cmd), "", "  ")
	return "保存前完整资料 / Full before:\n" + string(before) + "\n\n保存后完整资料 / Full after (Host will set updatedAt):\n" + string(after)
}
func roomEditView(c *ui.Context, m RoomEditModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("native-room-edit")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "群名、说明、成员与限额 / Group profile and limits").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "编辑已选群聊的名称、完整说明，以及原有成员顺序和限额。工作区、头像及其他资料保持原样；没有独立 title 输入。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, "读取群聊资料 / Read group profile").Disabled(blocked).Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "Host warning: "+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读取完整资料与 revision；旧草稿不会自动覆盖 Host / Read complete profile and revision first").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Loaded && !s.Writable {
			ui.Text(c, "Host 写权限未核实或目录只读 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, "无法证明未编辑资料可无损保留，原生保存已禁用；请使用现有 Bot Web 入口。\n"+s.Status.WriteError).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Loaded {
			r := s.Status.Item
			ui.Text(c, fmt.Sprintf("Host: %s · revision %d\nGroup: %s (%s)\nCanonical session: %s\nWorkspace: %s\n当前轮数 / Rounds: %d · 发言上限 / Speaks: %d\n当前成员（有序） / Current members:\n%s", m.HostLabel, s.Status.Catalog.Revision, r.Name, r.ID, r.SessionID, r.WorkspaceID, r.MaxRounds, r.MaxSpeaks, roomEditMembers(r.MemberBotIDs, s.Status.Catalog))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, "编辑群聊设置 / Edit group settings").Disabled(blocked || !s.Loaded || !s.Writable || s.Status.WriteError != "" || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		disabled := blocked || s.Uncertain || s.Review != nil
		roomEditTextInputs(c, m, disabled)
		for _, f := range []struct{ id, label, value string }{{"maxRounds", "轮数 / Rounds (1–3)", s.Draft.MaxRounds}, {"maxSpeaks", "成员发言上限 / Speaks (1–10)", s.Draft.MaxSpeaks}} {
			f := f
			v := f.value
			ui.TextInput(c.Key("room-edit-"+f.id), &v).Label(f.label).FillWidth().Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, v)
				}
			})
		}
		selected := map[string]bool{}
		for _, id := range s.Draft.MemberBotIDs {
			selected[id] = true
		}
		ui.Text(c, "成员保持选择顺序；移除后再选择会排到末尾 / Removing and selecting again moves a member to the end").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		for _, row := range s.Status.Catalog.Items {
			row := row
			if row.Kind != "bot" {
				continue
			}
			checked := selected[row.ID]
			label := fmt.Sprintf("%s · Bot %s · Session %s", row.Name, row.ID, row.SessionID)
			if row.Hidden {
				label += " (hidden)"
			}
			if ui.Checkbox(c.Key("room-edit-member-"+row.ID), &checked, label).Disabled(disabled || !s.Loaded).Changed() && m.OnMember != nil {
				m.OnMember(row.ID)
			}
		}
		for _, id := range s.Draft.MemberBotIDs {
			available := false
			for _, row := range s.Status.Catalog.Items {
				if row.ID == id && row.Kind == "bot" {
					available = true
				}
			}
			if !available {
				id := id
				checked := true
				if ui.Checkbox(c.Key("room-edit-missing-"+id), &checked, "移除不可用成员 / Remove unavailable member: "+id).Disabled(disabled).Changed() && m.OnMember != nil {
					m.OnMember(id)
				}
			}
		}
		ui.Text(c, "草稿成员顺序 / Draft order:\n"+roomEditMembers(s.Draft.MemberBotIDs, s.Status.Catalog)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
		if s.Uncertain {
			ui.Text(c, "结果未知，草稿保持锁定；目录不存在或不同也不自动重发。关闭界面不撤销已发送更新。本地保护只在当前进程保留；重启后请先通过 Web 核对。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if s.Candidate && m.OnAdopt != nil && ui.Button(c, "采用完整匹配的读回 / Adopt exact readback").Disabled(blocked).Clicked() {
				m.OnAdopt()
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, "丢弃群聊编辑草稿 / Discard edit draft").Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, "审阅群聊变更 / Review group changes").Disabled(disabled || !s.Loaded || !s.Writable || s.Status.WriteError != "").Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			cmd := r.Command
			before, after := cmd.Expected.Item, cmd.Values
			ui.Column(c.Key("room-edit-confirmation")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, "保存前完整审阅 / Review group update").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s · revision %d\nGroup: %s (%s) · Session: %s\n轮数 / Rounds: %d → %d\n发言上限 / Speaks: %d → %d\n原成员 / Before members:\n%s\n新成员 / After members:\n%s", m.HostLabel, cmd.Expected.Catalog.Revision, before.Name, before.ID, before.SessionID, before.MaxRounds, after.MaxRounds, before.MaxSpeaks, after.MaxSpeaks, roomEditMembers(before.MemberBotIDs, cmd.Expected.Catalog), roomEditMembers(after.MemberBotIDs, cmd.Expected.Catalog))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "Host 会写入目录和审计记录，更新原会话展示并尝试同步会话标题。新成员与限额影响群聊后续读取配置时的执行，成员可能调用模型、工具或外部消息并产生费用；不保证正在进行的讨论立刻采用新设置。本次保存不发送提示词、不启动或停止群聊。工作区和 canonical session 不变。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				if cmd.Values.Text != nil {
					ui.Text(c, roomEditTextReview(cmd)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				ui.Text(c, roomEditReviewText(cmd)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, roomEditAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if m.OnDismiss != nil && ui.Button(c, "返回修改群聊设置 / Back to draft").Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, roomEditConfirmLabel).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
