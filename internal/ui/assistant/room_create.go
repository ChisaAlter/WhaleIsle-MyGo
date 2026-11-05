package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
	card := ui.Column(c.Key("native-room-create")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
	chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
	card.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCreateBotGroup)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantChoose26RealBotsFromTheCurrentHostC)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshCreationCatalog)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || s.Scope == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		widgets.Notice(c, "room_create:71", "", s.Notice != "", func() string { return s.Notice })
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadTheCurrentCatalogAndWriteAutho)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		} else if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable7)).TextColor(accent)
		}
		if s.Created.ID != "" {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantGroupSession), s.Created.Name, s.Created.ID, s.Created.SessionID)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, i18n.T(i18n.KeyW4AssistantOpeningOnlyFocusesThisGroupSDedicated)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if m.OnOpen != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantOpenCreatedGroup)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnOpen()
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantNewGroup)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
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
			ui.TextInput(c.Key("room-"+f.id), &value).Label(f.label).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(disabled || f.id == "scratchCwd" && d.WorkspaceID != "").OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, value)
				}
			})
		}
		createWorkspacePicker(c.Key("new-room-workspace"), d.WorkspaceID, m.Workspaces, m.WorkspacesKnown, disabled, m.OnChange, muted)
		description := d.Description
		ui.TextArea(c.Key("room-description"), &description).Label(i18n.T(i18n.KeyMiscAssistantDescription120002)).Lines(3, 8).Radius(8).Border(.5, border).FillWidth().Disabled(disabled).OnChange(func() {
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
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheCreationOutcomeIsUnknownTheLocalDr)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if row := s.Candidate; row != nil {
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantMatchingRecord), row.Name, row.ID, row.SessionID, row.Description, row.WorkspaceID, strings.Join(row.MemberBotIDs, ", "), row.MaxRounds, row.MaxSpeaks)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, i18n.T(i18n.KeyW4AssistantACatalogEntryDoesNotProveTheSessionW)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.OnAdopt != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantUseInspectedGroup)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
					m.OnAdopt(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardGroupDraft)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewGroupCreation)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(disabled || !s.Loaded || !s.Writable).Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			cmd := r.Command
			d := cmd.Draft
			ui.Column(c.Key("room-confirmation")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewNewGroup)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s · revision %d\nGroup: %s\nDescription: %s\nWorkspace ID: %s\nScratch cwd: %s\nRoom ID: %s\nCanonical session: %s\nRounds: %d · Speaks: %d\nMembers:\n%s", m.HostLabel, cmd.Expected.Revision, d.Name, d.Description, d.WorkspaceID, d.ScratchCwd, cmd.ID, cmd.SessionID, d.MaxRounds, d.MaxSpeaks, roomCreateMembers(cmd))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, i18n.T(i18n.KeyW4AssistantTheHostWritesTheRoomCatalogEntryAndA)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, i18n.T(i18n.KeyW4AssistantUsesTheHostDefaultAvatarGroupModelFie)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, roomCreateAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToGroupDraft)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, roomCreateConfirmLabel()).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
