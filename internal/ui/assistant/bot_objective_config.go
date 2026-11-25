package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
		if aButton(c, i18n.T(i18n.KeyMiscAssistantRefreshObjectiveConfiguration), blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != nil {
			ui.Text(c, s.Error.SentenceText()).TextColor(accent).Selectable()
		}
		widgets.AckLine(c, "bot_objective_config:41", s.Notice)
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
		aTextarea(c.Key("objective-config-text"), &draft.Objective, coss.TextareaOptions{Label: i18n.T(i18n.KeyMiscAssistantObjective), MaxLines: 10, Disabled: !editable}).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		aInput(c.Key("objective-config-rounds"), &draft.MaxRounds, coss.InputOptions{Label: i18n.T(i18n.KeyMiscAssistantRoundCap1200), Disabled: !editable}).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange(draft)
			}
		})
		if aButton(c, i18n.T(i18n.KeyMiscAssistantDiscardObjectiveDraft), blocked || !s.Loaded || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if aPrimary(c, i18n.T(i18n.KeyMiscAssistantReviewObjectiveConfiguration), !editable || !s.Dirty).Clicked() && m.OnReview != nil {
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
			if aCheckbox(c, &ack, botObjectiveConfigAcknowledgement(), blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if aButton(c, i18n.T(i18n.KeyMiscAssistantCancelObjectiveConfiguration), blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if aPrimary(c, i18n.T(i18n.KeyMiscAssistantConfirmObjectiveConfiguration), blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
