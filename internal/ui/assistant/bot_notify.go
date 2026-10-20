package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botNotifyToggleLabel() string {
	return i18n.T(i18n.KeyMiscAssistantAllowOriginalBotNotifications)
}
func botNotifyAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandNotificationScopeAndProf)
}

type BotNotifyModel struct {
	State                                                         dshbotstore.NotifySnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange, OnAcknowledge                                       func(bool)
}

func botNotifyValue(v bool) string {
	if v {
		return i18n.T(i18n.KeyMiscAssistantEnabled2)
	}
	return i18n.T(i18n.KeyMiscAssistantDisabled2)
}
func botNotifyBlock(c *ui.Context, m BotNotifyModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("bot-notify")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantPersonalBotNotificationSetting)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantTheOriginalBotWebConsumesNotifyWhenTh)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantThisEntryEditsSharedConfigurationThePa)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshNotificationSetting)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).TextColor(muted).Selectable()
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "Host warning: "+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadBackBeforeSaving3)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		current, err := pluginrpc.DshbotNotifyEnabled(s.Status)
		if err != nil {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantBooleanUnavailable)).TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantNotifyDetail), m.HostLabel, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botNotifyValue(current))).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil
		draft := s.Draft
		if ui.Checkbox(c, &draft, botNotifyToggleLabel()).Disabled(!editable).Changed() && m.OnChange != nil {
			m.OnChange(draft)
		}
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantDraft)+botNotifyValue(s.Draft)).TextColor(fg)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardNotificationDraft)).Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewNotificationSetting)).Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptMatchingNotificationReadback)).Disabled(blocked).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		before, _ := pluginrpc.DshbotNotifyEnabled(r.Command.Expected)
		ui.Column(c.Key("bot-notify-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantConfirmNotificationSetting)).Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantNotifyReviewDetail), m.HostLabel, r.Command.Expected.Bot.ID, r.Command.Expected.Bot.SessionID, r.Command.Expected.Revision, botNotifyValue(before), botNotifyValue(r.Command.Enabled))).TextColor(fg).Selectable()
			if r.Command.Enabled {
				ui.Text(c, i18n.T(i18n.KeyW4AssistantAllowsLaterQualifyingNotificationsItIs)).TextColor(accent)
			} else {
				ui.Text(c, i18n.T(i18n.KeyW4AssistantRemovesEligibilityForLaterNotifications)).TextColor(accent)
			}
			ui.Text(c, i18n.T(i18n.KeyW4AssistantAlthoughOnlyNotifyChangesTheFullBotUp)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, i18n.T(i18n.KeyW4AssistantABusyOrColdSessionMaySyncAtALaterI)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyW4AssistantKeepsHiddenNameAvatarModelWorkspaceAnd), botCapabilityProfileText(r.Command.Expected, "workspaceId"), botCapabilityProfileText(r.Command.Expected, "model"), botCapabilityProfileText(r.Command.Expected, "objective"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botNotifyAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelNotificationSave)).Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmNotificationSetting2)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
