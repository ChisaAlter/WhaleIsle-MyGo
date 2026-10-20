package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botObjectiveConfigAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandSavingMayStartOrResume)
}

type BotObjectiveConfigModel struct {
	State                                                dshbotstore.ObjectiveConfigSnapshot
	HostLabel                                            string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange                                             func(pluginrpc.DshbotObjectiveConfigDraft)
	OnAcknowledge                                        func(bool)
}

func botObjectiveConfigBlock(c *ui.Context, m BotObjectiveConfigModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && s.Review == nil
	ui.Column(c.Key("bot-objective-config")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantObjectiveConfiguration)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantAddOrChangeTheCatalogObjectiveOfAnEx)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshObjectiveConfiguration)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		for _, warning := range s.Warnings {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantHostWarning)+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadBackBeforeSavingAgain)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		current, err := pluginrpc.DshbotObjectiveConfigValue(s.Status)
		if err != nil {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheHostReturnedNoCompleteObjectiveFiel)).TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantCurrentObjectiveDetail), m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, current.Objective, current.MaxRounds)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable6)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		ui.TextArea(c.Key("objective-config-text"), &draft.Objective).Label(i18n.T(i18n.KeyMiscAssistantObjective)).FillWidth().Lines(3, 10).Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		ui.TextInput(c.Key("objective-config-rounds"), &draft.MaxRounds).Label(i18n.T(i18n.KeyMiscAssistantRoundCap1200)).FillWidth().Disabled(!editable).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardObjectiveDraft)).Disabled(blocked || !s.Loaded || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewObjectiveConfiguration)).Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		before := r.Command.Expected
		old, _ := pluginrpc.DshbotObjectiveConfigValue(before)
		ui.Column(c.Key("objective-config-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewObjectiveConfiguration2)).Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantObjectiveConfigDiff), m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, old.Objective, r.Command.Desired.Objective, old.MaxRounds, r.Command.Desired.MaxRounds)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantRetainedConfig), botCapabilityProfileText(before, "model"), botCapabilityProfileText(before, "workspaceId"), botCapabilityProfileText(before, "capabilities"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ui.Text(c, i18n.T(i18n.KeyW4AssistantThisBotUpdateSaveCallsRequestsyncALiv)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheSavePathDoesNotResolveAColdSessio)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, i18n.T(i18n.KeyW4AssistantOnlyObjectiveAndObjectivemaxroundsAreRe)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheSaveIsSentOnlyAfterYouSeparatelyC)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botObjectiveConfigAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelObjectiveConfiguration)).Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmObjectiveConfiguration)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
