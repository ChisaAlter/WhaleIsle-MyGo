package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func roomCreateAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandRoomAndSessionEffects)
}
func roomCreateConfirmLabel() string {
	return i18n.T(i18n.KeyMiscAssistantConfirmCreateAndOpen)
}

type RoomCreateModel struct {
	State                                                                 dshbotstore.RoomCreateSnapshot
	HostLabel                                                             string
	Workspaces                                                            []session.WorkspaceRow
	WorkspacesKnown                                                       bool
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
		name, session := i18n.T(i18n.KeyMiscAssistantUnavailable), ""
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
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCreateBotGroup)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "从当前 Host 目录选择 2–6 个真实 Bot，创建群聊资料和专属会话。这里只创建群聊，不编辑现有群组或直接运行讨论。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshCreationCatalog)).Disabled(blocked || s.Scope == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadTheCurrentCatalogAndWriteAutho)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable7)).TextColor(accent)
		}
		if s.Created.ID != "" {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantGroupSession), s.Created.Name, s.Created.ID, s.Created.SessionID)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, "打开只定位此群聊的专属会话；Host 可能恢复缺失会话。打开失败不会重新创建群聊。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if m.OnOpen != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantOpenCreatedGroup)).Disabled(blocked).Clicked() {
				m.OnOpen()
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantNewGroup)).Disabled(blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantSelectedMembers), m.HostLabel, s.Catalog.Revision, len(d.MemberBotIDs))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		for _, f := range []struct{ id, label, value string }{{"name", i18n.T(i18n.KeyMiscAssistantGroupName120), d.Name}, {"scratchCwd", i18n.T(i18n.KeyMiscAssistantScratchCwdRequired), d.ScratchCwd}, {"maxRounds", i18n.T(i18n.KeyMiscAssistantRounds13), d.MaxRounds}, {"maxSpeaks", i18n.T(i18n.KeyMiscAssistantSpeaks110), d.MaxSpeaks}} {
			f := f
			value := f.value
			ui.TextInput(c.Key("room-"+f.id), &value).Label(f.label).FillWidth().Disabled(disabled || f.id == "scratchCwd" && d.WorkspaceID != "").OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, value)
				}
			})
		}
		createWorkspacePicker(c.Key("new-room-workspace"), d.WorkspaceID, m.Workspaces, m.WorkspacesKnown, disabled, m.OnChange, muted)
		description := d.Description
		ui.TextArea(c.Key("room-description"), &description).Label(i18n.T(i18n.KeyMiscAssistantDescription120002)).Lines(3, 8).FillWidth().Disabled(disabled).OnChange(func() {
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
				label += i18n.T(i18n.KeyMiscAssistantHidden)
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
				if ui.Checkbox(c.Key("room-missing-"+id), &checked, i18n.T(i18n.KeyMiscAssistantRemoveUnavailableMember)+id).Disabled(disabled).Changed() && m.OnMember != nil {
					m.OnMember(id)
				}
			}
		}
		if s.Uncertain {
			ui.Text(c, "创建结果未知：本地草稿与身份保持锁定，不自动重试。即使读回没找到记录也不重建；Host 没有请求幂等保证。关闭界面不取消已发送操作，本地保护只在当前应用进程保留。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if row := s.Candidate; row != nil {
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantMatchingRecord), row.Name, row.ID, row.SessionID, row.Description, row.WorkspaceID, strings.Join(row.MemberBotIDs, ", "), row.MaxRounds, row.MaxSpeaks)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "目录存在不证明会话已成功建立；采用后再打开会核实或恢复会话，不调用 bot/create。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.OnAdopt != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantUseInspectedGroup)).Disabled(blocked).Clicked() {
					m.OnAdopt(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardGroupDraft)).Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewGroupCreation)).Disabled(disabled || !s.Loaded || !s.Writable).Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			cmd := r.Command
			d := cmd.Draft
			ui.Column(c.Key("room-confirmation")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewNewGroup)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s · revision %d\nGroup: %s\nDescription: %s\nWorkspace ID: %s\nScratch cwd: %s\nRoom ID: %s\nCanonical session: %s\nRounds: %d · Speaks: %d\nMembers:\n%s", m.HostLabel, cmd.Expected.Revision, d.Name, d.Description, d.WorkspaceID, d.ScratchCwd, cmd.ID, cmd.SessionID, d.MaxRounds, d.MaxSpeaks, roomCreateMembers(cmd))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "Host 将写入群聊目录与审计记录，创建 dshbot-room 专属会话并打开它；可能启动相关会话生命周期处理。群聊后续使用成员 Bot 的模型/工具，可能产生费用或外部消息。本次不发送讨论提示词、不运行群聊。非空 workspace 优先于 scratch cwd。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, "使用 Host 默认头像；群聊模型字段为空，tools/skills/MCP 为全部，allowedSenderIds 为空，无目标或例行任务。创建结果未知时保留并锁定草稿，仅只读核对，不自动重试；打开失败与创建失败分开处理。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, roomCreateAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToGroupDraft)).Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, roomCreateConfirmLabel()).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
