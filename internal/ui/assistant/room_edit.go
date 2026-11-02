package assistant

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func roomEditAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandGroupProfileAndSessionE)
}
func roomEditConfirmLabel() string {
	return i18n.T(i18n.KeyMiscAssistantConfirmGroupUpdate)
}

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
		name, session := i18n.T(i18n.KeyMiscAssistantUnavailable), ""
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
	return i18n.T(i18n.KeyMiscAssistantFullBefore) + string(before) + i18n.T(i18n.KeyMiscAssistantFullAfter) + string(after)
}
func roomEditView(c *ui.Context, m RoomEditModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	card := ui.Column(c.Key("native-room-edit")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
	chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
	card.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantGroupProfileAndLimits)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantEditTheSelectedGroupSNameFullDescript)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReadGroupProfile)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
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
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadCompleteProfileAndRevisionFirst)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		if s.Loaded && !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable3)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantLosslessPreservationOfUneditedDataCanno)+s.Status.WriteError).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Loaded {
			r := s.Status.Item
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantRoomDetail), m.HostLabel, s.Status.Catalog.Revision, r.Name, r.ID, r.SessionID, r.WorkspaceID, r.MaxRounds, r.MaxSpeaks, roomEditMembers(r.MemberBotIDs, s.Status.Catalog))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantEditGroupSettings)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || !s.Writable || s.Status.WriteError != "" || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		disabled := blocked || s.Uncertain || s.Review != nil
		roomEditTextInputs(c, m, disabled)
		for _, f := range []struct{ id, label, value string }{{"maxRounds", i18n.T(i18n.KeyMiscAssistantRounds13), s.Draft.MaxRounds}, {"maxSpeaks", i18n.T(i18n.KeyMiscAssistantSpeaks110), s.Draft.MaxSpeaks}} {
			f := f
			v := f.value
			ui.TextInput(c.Key("room-edit-"+f.id), &v).Label(f.label).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, v)
				}
			})
		}
		selected := map[string]bool{}
		for _, id := range s.Draft.MemberBotIDs {
			selected[id] = true
		}
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantRemovingAndSelectingAgainMovesAMem)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
				if ui.Checkbox(c.Key("room-edit-missing-"+id), &checked, i18n.T(i18n.KeyMiscAssistantRemoveUnavailableMember2)+id).Disabled(disabled).Changed() && m.OnMember != nil {
					m.OnMember(id)
				}
			}
		}
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantDraftOrder)+roomEditMembers(s.Draft.MemberBotIDs, s.Status.Catalog)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
		if s.Uncertain {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheOutcomeIsUnknownAndTheDraftStaysL)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if s.Candidate && m.OnAdopt != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptExactReadback2)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnAdopt()
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardEditDraft)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewGroupChanges)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(disabled || !s.Loaded || !s.Writable || s.Status.WriteError != "").Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			cmd := r.Command
			before, after := cmd.Expected.Item, cmd.Values
			ui.Column(c.Key("room-edit-confirmation")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewGroupUpdate)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantRoomEditDiff), m.HostLabel, cmd.Expected.Catalog.Revision, before.Name, before.ID, before.SessionID, before.MaxRounds, after.MaxRounds, before.MaxSpeaks, after.MaxSpeaks, roomEditMembers(before.MemberBotIDs, cmd.Expected.Catalog), roomEditMembers(after.MemberBotIDs, cmd.Expected.Catalog))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, i18n.T(i18n.KeyW4AssistantTheHostWritesCatalogAndAuditRecordsUp)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				if cmd.Values.Text != nil {
					ui.Text(c, roomEditTextReview(cmd)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				ui.Text(c, roomEditReviewText(cmd)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, roomEditAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToDraft)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, roomEditConfirmLabel()).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
