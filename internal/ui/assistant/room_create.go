package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
)

const roomCreateAcknowledgement = "我理解创建会话与后续群聊执行影响 / I understand room and session effects"
const roomCreateConfirmLabel = "确认创建并打开群聊 / Confirm create and open"

type RoomCreateModel struct {
	State                                                                 dshbotstore.RoomCreateSnapshot
	HostLabel                                                             string
	OnRefresh, OnStart, OnReview, OnConfirm, OnDismiss, OnDiscard, OnOpen func()
	OnChange                                                              func(field, value string)
	OnMember                                                              func(id string)
	OnAcknowledge                                                         func(bool)
	OnAdopt                                                               func(id string)
}

func roomCreateBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.RoomCreate.OnRefresh == nil {
		return
	}
	roomCreateView(c, m.RoomCreate, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func roomCreateMembers(cmd pluginrpc.DshbotRoomCreateCommand) string {
	lines := []string{}
	for _, id := range cmd.Draft.MemberBotIDs {
		name, session := "不可用 / unavailable", ""
		for _, row := range cmd.Expected.Items {
			if row.ID == id && row.Kind == "bot" {
				name = row.Name
				session = row.SessionID
				break
			}
		}
		lines = append(lines, fmt.Sprintf("%s · Bot %s · Session %s", name, id, session))
	}
	return strings.Join(lines, "\n")
}
func roomCreateView(c *ui.Context, m RoomCreateModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("native-room-create")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "新建 Bot 群聊 / Create Bot group").FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "从当前 Host 目录选择 2–6 个真实 Bot，创建群聊资料和专属会话。这里只创建群聊，不编辑现有群组或直接运行讨论。").FontSize(11).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, "刷新群聊创建目录 / Refresh creation catalog").Disabled(blocked || s.Scope == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(12).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读取完整目录、revision 与写权限 / Read the current catalog and write authority first").FontSize(11).TextColor(muted)
		} else if !s.Writable {
			ui.Text(c, "Host 目录只读或写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Created.ID != "" {
			ui.Text(c, fmt.Sprintf("群聊 / Group: %s (%s)\nCanonical session: %s", s.Created.Name, s.Created.ID, s.Created.SessionID)).FontSize(12).TextColor(fg).Selectable()
			ui.Text(c, "打开只定位此群聊的专属会话；Host 可能恢复缺失会话。打开失败不会重新创建群聊。").FontSize(11).TextColor(muted)
			if m.OnOpen != nil && ui.Button(c, "打开已有群聊 / Open created group").Disabled(blocked).Clicked() {
				m.OnOpen()
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, "新建群聊 / New group").Disabled(blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		ui.Text(c, fmt.Sprintf("Host: %s · revision %d\n已选成员 / Selected members: %d (2–6)", m.HostLabel, s.Catalog.Revision, len(d.MemberBotIDs))).FontSize(11).TextColor(muted)
		for _, f := range []struct{ id, label, value string }{{"name", "群聊名称 / Group name (≤120)", d.Name}, {"workspaceId", "工作区 ID / Workspace ID (空=临时目录)", d.WorkspaceID}, {"scratchCwd", "临时会话目录 / Scratch cwd (无工作区时必填)", d.ScratchCwd}, {"maxRounds", "轮数 / Rounds (1–3)", d.MaxRounds}, {"maxSpeaks", "成员发言上限 / Speaks (1–10)", d.MaxSpeaks}} {
			f := f
			value := f.value
			ui.TextInput(c.Key("room-"+f.id), &value).Label(f.label).FillWidth().Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, value)
				}
			})
		}
		description := d.Description
		ui.TextArea(c.Key("room-description"), &description).Label("群聊说明 / Description (≤12000)").Lines(3, 8).FillWidth().Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("description", description)
			}
		})
		selected := map[string]bool{}
		for _, id := range d.MemberBotIDs {
			selected[id] = true
		}
		for _, row := range s.Catalog.Items {
			row := row
			if row.Kind != "bot" {
				continue
			}
			checked := selected[row.ID]
			label := fmt.Sprintf("%s · %s · %s", row.Name, row.ID, row.SessionID)
			if row.Hidden {
				label += " (已隐藏 / hidden)"
			}
			if ui.Checkbox(c.Key("room-member-"+row.ID), &checked, label).Disabled(disabled || !s.Loaded).Changed() && m.OnMember != nil {
				m.OnMember(row.ID)
			}
		}
		for _, id := range d.MemberBotIDs {
			available := false
			for _, row := range s.Catalog.Items {
				if row.ID == id && row.Kind == "bot" {
					available = true
				}
			}
			if !available {
				id := id
				checked := true
				if ui.Checkbox(c.Key("room-missing-"+id), &checked, "已选成员已不可用，请移除 / Remove unavailable member: "+id).Disabled(disabled).Changed() && m.OnMember != nil {
					m.OnMember(id)
				}
			}
		}
		if s.Uncertain {
			ui.Text(c, "创建结果未知：本地草稿与身份保持锁定，不自动重试。即使读回没找到记录也不重建；Host 没有请求幂等保证。关闭界面不取消已发送操作，本地保护只在当前应用进程保留。").FontSize(12).TextColor(accent)
			if row := s.Candidate; row != nil {
				ui.Text(c, fmt.Sprintf("同身份目录记录 / Matching record: %s (%s)\nSession: %s\nDescription: %s\nWorkspace: %s\nMembers: %s\nRounds: %d · Speaks: %d", row.Name, row.ID, row.SessionID, row.Description, row.WorkspaceID, strings.Join(row.MemberBotIDs, ", "), row.MaxRounds, row.MaxSpeaks)).FontSize(12).TextColor(fg).Selectable()
				ui.Text(c, "目录存在不证明会话已成功建立；采用后再打开会核实或恢复会话，不调用 bot/create。").FontSize(11).TextColor(muted)
				if m.OnAdopt != nil && ui.Button(c, "采用核对后的群聊 / Use inspected group").Disabled(blocked).Clicked() {
					m.OnAdopt(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, "丢弃群聊草稿 / Discard group draft").Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, "审阅群聊创建 / Review group creation").Disabled(disabled || !s.Loaded || !s.Writable).Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			cmd := r.Command
			d := cmd.Draft
			ui.Column(c.Key("room-confirmation")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, "创建前完整审阅 / Review new group").FontSize(13).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s · revision %d\nGroup: %s\nDescription: %s\nWorkspace ID: %s\nScratch cwd: %s\nRoom ID: %s\nCanonical session: %s\nRounds: %d · Speaks: %d\nMembers:\n%s", m.HostLabel, cmd.Expected.Revision, d.Name, d.Description, d.WorkspaceID, d.ScratchCwd, cmd.ID, cmd.SessionID, d.MaxRounds, d.MaxSpeaks, roomCreateMembers(cmd))).FontSize(12).TextColor(fg).Selectable()
				ui.Text(c, "Host 将写入群聊目录与审计记录，创建 dshbot-room 专属会话并打开它；可能启动相关会话生命周期处理。群聊后续使用成员 Bot 的模型/工具，可能产生费用或外部消息。本次不发送讨论提示词、不运行群聊。非空 workspace 优先于 scratch cwd。").FontSize(12).TextColor(accent)
				ui.Text(c, "使用 Host 默认头像；群聊模型字段为空，tools/skills/MCP 为全部，allowedSenderIds 为空，无目标或例行任务。创建结果未知时保留并锁定草稿，仅只读核对，不自动重试；打开失败与创建失败分开处理。").FontSize(11).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, roomCreateAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if m.OnDismiss != nil && ui.Button(c, "返回修改群聊 / Back to group draft").Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, roomCreateConfirmLabel).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
