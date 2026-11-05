package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type RoomRuntimeModel struct {
	State     dshbotstore.RoomRuntimeSnapshot
	HostLabel string
	OnRefresh func()
	OnOpen    func(sessionID string)
}

func roomRuntimeOpenLabel() string {
	return i18n.T(i18n.KeyMiscAssistantOpenMemberSession)
}

func roomRuntimeBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.RoomRuntime.State.Scope == "" || m.RoomRuntime.OnRefresh == nil {
		return
	}
	roomRuntimeView(c, m.RoomRuntime, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func roomRuntimeView(c *ui.Context, m RoomRuntimeModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	card := ui.Column(c.Key("native-room-runtime")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
	chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
	card.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantGroupMemberRuntime)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantReadsCompleteMemberStatesErrorsAndPend)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshMemberRuntime)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRuntimeFailure)+s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		widgets.Notice(c, "room_runtime:47", "", s.Notice != "", func() string { return s.Notice })
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantCurrentRuntimeUnverifiedAnyPrevious)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		}
		if s.Data.Room.ID == "" {
			return
		}
		room := s.Data.Room
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyW4AssistantHostSNroomSSCanonicalSessionSNcatalo), m.HostLabel, room.Name, room.ID, room.SessionID, s.Data.Catalog.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		if len(s.Data.Members) == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoVisibleMemberRuntimeThisIsNotA)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		for i, member := range s.Data.Members {
			member := member
			label := member.Name
			if strings.TrimSpace(label) == "" {
				label = i18n.T(i18n.KeyMiscAssistantUnknownMember)
			}
			status := member.Status
			if status == "" {
				status = i18n.T(i18n.KeyMiscAssistantUnknown)
			}
			memCard := ui.Column(c.Key(fmt.Sprintf("room-runtime-member-%d", i))).Border(.5, border).Radius(12).Padding(10, 12).Gap(6)
			chrome.GlassSurface(memCard, field, chrome.MaterialPanel, 12)
			memCard.Children(func() {
				ui.Text(c, label+" · Bot "+member.BotID).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				ui.Text(c, fmt.Sprintf("Room session: %s\nMember session: %s\nHost status: %s", member.RoomSessionID, member.SessionID, status)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				if pluginrpc.DshbotRoomRuntimeAttention(member) {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantNeedsAttention)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				}
				if len(member.LastError) > 0 {
					ui.Text(c, "lastError:\n"+pluginrpc.DshbotRoomRuntimeJSON(member.LastError)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				if len(member.Error) > 0 {
					ui.Text(c, "error:\n"+pluginrpc.DshbotRoomRuntimeJSON(member.Error)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				if len(member.Pending) > 0 {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantPendingItems)+pluginrpc.DshbotRoomRuntimeJSON(member.Pending)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				if member.OpenError != "" {
					ui.Text(c, member.OpenError).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent).Selectable()
				}
				ui.Text(c, i18n.T(i18n.KeyW4AssistantOpeningSwitchesToTheVerifiedExistingMe)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.OnOpen != nil && ui.Button(c, roomRuntimeOpenLabel()).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || member.OpenError != "" || member.SessionID == "").Clicked() {
					m.OnOpen(member.SessionID)
				}
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantCompleteMemberRecord)+pluginrpc.DshbotRoomRuntimeJSON(member.Raw)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			})
		}
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCompleteRuntimeResponse)+pluginrpc.DshbotRoomRuntimeJSON(s.Data.Raw)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
	})
}
