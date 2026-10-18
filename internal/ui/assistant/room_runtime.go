package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

type RoomRuntimeModel struct {
	State     dshbotstore.RoomRuntimeSnapshot
	HostLabel string
	OnRefresh func()
	OnOpen    func(sessionID string)
}

const roomRuntimeOpenLabel = "打开此成员会话 / Open member session"

func roomRuntimeBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.RoomRuntime.State.Scope == "" || m.RoomRuntime.OnRefresh == nil {
		return
	}
	roomRuntimeView(c, m.RoomRuntime, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func roomRuntimeView(c *ui.Context, m RoomRuntimeModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("native-room-runtime")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "群聊成员运行状态 / Group member runtime").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "读取完整成员状态、错误与待处理事项。刷新只读，不启动讨论、跟随成员或自动回答。开始讨论仍使用群聊会话发送；整群停止使用现有确认入口。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, "刷新成员运行状态 / Refresh member runtime").Disabled(blocked).Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, "group/runtime 读取或打开失败 / Runtime or open failure:\n"+s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "当前状态未核实，已有快照也不是当前运行结果 / Current runtime unverified; any previous snapshot is stale").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		}
		if s.Data.Room.ID == "" {
			return
		}
		room := s.Data.Room
		ui.Text(c, fmt.Sprintf("Host: %s\nRoom: %s (%s) · canonical session: %s\n目录 revision: %d · 运行状态为独立读取快照", m.HostLabel, room.Name, room.ID, room.SessionID, s.Data.Catalog.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		if len(s.Data.Members) == 0 {
			ui.Text(c, "没有可见成员运行记录；不代表成员已完成或健康 / No visible member runtime; this is not a completion or health receipt").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		for i, member := range s.Data.Members {
			member := member
			label := member.Name
			if strings.TrimSpace(label) == "" {
				label = "身份未知 / Unknown member"
			}
			status := member.Status
			if status == "" {
				status = "未知 / unknown"
			}
			ui.Column(c.Key(fmt.Sprintf("room-runtime-member-%d", i))).Background(field).Border(1, border).Radius(6).Padding(10).Gap(6).Children(func() {
				ui.Text(c, label+" · Bot "+member.BotID).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Room session: %s\nMember session: %s\nHost status: %s", member.RoomSessionID, member.SessionID, status)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
				if pluginrpc.DshbotRoomRuntimeAttention(member) {
					ui.Text(c, "需要留意 / Needs attention").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				}
				if len(member.LastError) > 0 {
					ui.Text(c, "lastError:\n"+pluginrpc.DshbotRoomRuntimeJSON(member.LastError)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				if len(member.Error) > 0 {
					ui.Text(c, "error:\n"+pluginrpc.DshbotRoomRuntimeJSON(member.Error)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				if len(member.Pending) > 0 {
					ui.Text(c, "待处理 / Pending:\n"+pluginrpc.DshbotRoomRuntimeJSON(member.Pending)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				if member.OpenError != "" {
					ui.Text(c, member.OpenError).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent).Selectable()
				}
				ui.Text(c, "打开将切换到已核实的既有成员会话并观察其历史/交互；可能恢复该会话。不会创建新会话、发送提示词或自动批准/回答。后续处理由你在会话中明确操作。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.OnOpen != nil && ui.Button(c, roomRuntimeOpenLabel).Disabled(blocked || !s.Loaded || member.OpenError != "" || member.SessionID == "").Clicked() {
					m.OnOpen(member.SessionID)
				}
				ui.Text(c, "完整成员记录 / Complete member record:\n"+pluginrpc.DshbotRoomRuntimeJSON(member.Raw)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			})
		}
		ui.Text(c, "完整运行回包（含扩展字段） / Complete runtime response:\n"+pluginrpc.DshbotRoomRuntimeJSON(s.Data.Raw)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
	})
}
