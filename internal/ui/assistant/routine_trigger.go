package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func routineTriggerAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIApproveObtainingPersistentHostCata)
}
func routineTriggerImpact() string {
	return i18n.T(i18n.KeyW4AssistantThisCallsTheOriginalBotTriggerTokenAn)
}

func routineTriggerBlock(c *ui.Context, m RoutineModel, blocked bool, fg, muted, accent ui.Color) {
	s := m.State.Trigger
	if m.OnPrepareTrigger == nil && m.TriggerUnavailable == "" {
		return
	}
	ui.Column(c.Key("routine-trigger-access")).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCurrentHostBotCatalogTriggerCredent)).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
		if m.TriggerUnavailable != "" {
			ui.Text(c, m.TriggerUnavailable).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			return
		}
		if !s.Open {
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewHostCatalogCredential)).Disabled(blocked || !m.State.Loaded).Clicked() {
				m.OnPrepareTrigger()
			}
			ui.Text(c, i18n.T(i18n.KeyW4AssistantOpeningThePageOrRefreshingTheCatalogD)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s\nRoutine: %s (%s) · revision %d", m.HostLabel, m.State.BotName, m.State.Scope.BotID, s.SessionID, s.RoutineName, s.RoutineID, s.Revision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if s.Review {
			ui.Text(c, routineTriggerImpact()).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheConfirmationBindsToTheCurrentlyAuth)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ack := s.Acknowledged
			if ui.Checkbox(c, &ack, routineTriggerAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledgeTrigger != nil {
				m.OnAcknowledgeTrigger(ack)
			}
			if m.OnConfirmTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmHostCatalogCredential)).Disabled(blocked || !ack || !m.State.Loaded).Clicked() {
				m.OnConfirmTrigger()
			}
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		}
		if s.Ready && m.TriggerMaterial != nil {
			address, header := m.TriggerMaterial()
			if address != "" && header != "" {
				ui.Text(c, "POST "+address).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, header).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, i18n.T(i18n.KeyW4AssistantASensitiveSharedCredentialShownOnlyTem)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
				ui.Row(c).Gap(8).Children(func() {
					if m.OnCopyTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCopyTriggerAddress)).Disabled(blocked).Clicked() {
						m.OnCopyTrigger("address")
					}
					if m.OnCopyTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCopySecretHeader)).Disabled(blocked).Clicked() {
						m.OnCopyTrigger("header")
					}
				})
			}
		}
		if m.OnCloseTrigger != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCloseAndClearCredential)).Clicked() {
			m.OnCloseTrigger()
		}
	})
}
