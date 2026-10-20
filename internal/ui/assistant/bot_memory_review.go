package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botMemoryReviewAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandModelMemoryAndObjective)
}

type BotMemoryReviewModel struct {
	State                                                dshbotstore.MemoryReviewSnapshot
	HostLabel                                            string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange, OnAcknowledge                              func(bool)
}

func botMemoryReviewValue(enabled bool) string {
	if enabled {
		return i18n.T(i18n.KeyMiscAssistantEnabled)
	}
	return i18n.T(i18n.KeyMiscAssistantDisabled)
}
func botMemoryReviewBlock(c *ui.Context, m BotMemoryReviewModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && s.Review == nil
	ui.Column(c.Key("bot-memory-review")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantAutomaticMemoryLearning)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantManagesTheOriginalMemoryreviewSettingA)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshMemoryLearning)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
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
		current, err := pluginrpc.DshbotMemoryReviewEnabled(s.Status)
		if err != nil {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheHostReturnedNoValidMemoryreviewNoD)).TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyW4AssistantHostSNbotSSessionSRevisionDNhostCon), m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botMemoryReviewValue(current))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable6)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		if ui.Checkbox(c, &draft, i18n.T(i18n.KeyMiscAssistantAllowAutomaticMemoryLearning)).Disabled(!editable).Changed() && m.OnChange != nil {
			m.OnChange(draft)
		}
		ui.Text(c, i18n.T(i18n.KeyW4AssistantOnceEnabledTheSideModelCanReadRecent)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantDisablingOnlyBlocksNewChecksItDoesNot)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardMemoryLearningDraft)).Disabled(blocked || !s.Loaded || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewMemoryLearning)).Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Review != nil {
			r := s.Review
			before := r.Command.Expected
			old, _ := pluginrpc.DshbotMemoryReviewEnabled(before)
			ui.Column(c.Key("memory-review-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewMemoryLearning2)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantMemoryReviewDetail), m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, botMemoryReviewValue(old), botMemoryReviewValue(r.Command.Enabled), botCapabilityProfileText(before, "model"), botCapabilityProfileText(before, "objective"), botCapabilityProfileText(before, "objectiveMaxRounds"))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				if r.Command.Enabled {
					ui.Text(c, i18n.T(i18n.KeyW4AssistantAllowsLaterEventsToInvokeTheSideModel)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				} else {
					ui.Text(c, i18n.T(i18n.KeyW4AssistantBlocksNewAutomaticChecksItDoesNotCanc)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				}
				ui.Text(c, i18n.T(i18n.KeyW4AssistantThisSaveOnlyChangesMemoryreviewOtherDa)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, i18n.T(i18n.KeyW4AssistantThisPageOffersNoImmediateCheckOrReset)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botMemoryReviewAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelMemoryLearning)).Disabled(blocked).Clicked() && m.OnDismiss != nil {
					m.OnDismiss()
				}
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmMemoryLearning)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
					m.OnConfirm()
				}
			})
		}
	})
}
