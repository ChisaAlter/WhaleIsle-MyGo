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
		ui.Text(c, i18n.T(i18n.KeyW4AssistantNameIsTheStableAddressNameRequiredUp)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
			ui.Text(c, i18n.T(i18n.KeyW4AssistantNewCompleteDescriptionPersonaTrimmedAft)+botTextProfileOptional(cmd.Values.Description)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			if old.Name != cmd.Values.Name {
				ui.Text(c, i18n.T(i18n.KeyW4AssistantChangingTheAddressNameMayBreakOldAddr)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			}
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTitleTakesPriorityForTheDisplayNameBu)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, i18n.T(i18n.KeyW4AssistantThisBotUpdateSaveWritesAuditRecordsUp)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, i18n.T(i18n.KeyW4AssistantAColdSessionIsNotResolvedForObjective)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyW4AssistantKeepsWorkspaceidSNkeepsModelSNonlyNam), botCapabilityProfileText(cmd.Expected, "workspaceId"), botCapabilityProfileText(cmd.Expected, "model"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
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
