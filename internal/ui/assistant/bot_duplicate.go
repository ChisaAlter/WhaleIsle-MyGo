package assistant

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botDuplicateAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIConfirmThisSourceAndCopyEffects)
}
func botDuplicateExecutionAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIConfirmCopiedGoalExecutionEffects)
}
func botDuplicateConfirmLabel() string {
	return i18n.T(i18n.KeyMiscAssistantConfirmDuplicateBot)
}

type BotDuplicateModel struct {
	State                                                         dshbotstore.BotDuplicateSnapshot
	HostLabel, SourceID                                           string
	OnRefresh, OnStart, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange                                                      func(field, value string)
	OnAcknowledge, OnAcknowledgeExecution                         func(bool)
	OnOpen                                                        func(id string)
}

func botDuplicateBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.BotDuplicate.OnRefresh == nil {
		return
	}
	botDuplicateView(c, m.BotDuplicate, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func botDuplicateConfigText(item pluginrpc.DshbotBotItem) string {
	config := map[string]json.RawMessage{}
	for _, key := range pluginrpc.DshbotBotDuplicateConfigKeys {
		if raw, ok := item.Profile[key]; ok {
			config[key] = raw
		}
	}
	raw, _ := json.MarshalIndent(config, "", "  ")
	return string(raw)
}
func botDuplicateReviewText(host string, cmd pluginrpc.DshbotBotDuplicateCommand) string {
	var source pluginrpc.DshbotBotItem
	for _, item := range cmd.Expected.Items {
		if item.ID == cmd.SourceID {
			source = item
			break
		}
	}
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantBotDuplicateConfig), host, cmd.Expected.Revision, source.Name, cmd.SourceID, source.SessionID, cmd.Name, cmd.ScratchCwd, botDuplicateConfigText(source))
}
func botDuplicateView(c *ui.Context, m BotDuplicateModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	card := ui.Column(c.Key("native-bot-duplicate")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
	chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
	card.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantDuplicateSelectedPersonalBot)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantCopiesTheFullConfigurationTheOriginalS)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReadDuplicateSource)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || s.Scope == "" || m.SourceID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		for _, warning := range s.Warnings {
			ui.Text(c, "Host warning: "+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadTheFullCurrentSourceAndWriteA)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		} else if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable5)).TextColor(accent)
		}
		if s.Created.ID != "" {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantReceiptLinkedCopy), s.Created.Name, s.Created.ID, s.Created.SessionID, botDuplicateConfigText(s.Created))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheProfileAndSessionReceiptsAreConfirm)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			if m.OnOpen != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantOpenReceiptLinkedCopy)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnOpen(s.Created.ID)
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantPrepareDuplicate)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		name := d.Name
		ui.TextInput(c.Key("bot-duplicate-name"), &name).Label(i18n.T(i18n.KeyMiscAssistantNewStableName120)).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("name", name)
			}
		})
		cwd := d.ScratchCwd
		ui.TextInput(c.Key("bot-duplicate-cwd"), &cwd).Label(i18n.T(i18n.KeyMiscAssistantScratchCwdOverridden)).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("scratchCwd", cwd)
			}
		})
		if s.Uncertain {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheCopyOutcomeStaysUnknownAndCannotBe)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if s.Pending != nil {
				ui.Text(c, botDuplicateReviewText(m.HostLabel, *s.Pending)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
			}
			if len(s.Candidates) == 0 {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingCandidateDoesNotEstablish)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
			for _, row := range s.Candidates {
				row := row
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantUnattributedCandidate), row.Name, row.ID, row.SessionID, botDuplicateConfigText(row))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, i18n.T(i18n.KeyW4AssistantReviewThisCatalogBotSFullConfiguration)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
				if m.OnOpen != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantOpenInspectedCandidate)+row.ID).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
					m.OnOpen(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardDuplicateDraft)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewDuplicate)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(disabled || !s.Loaded || !s.Writable).Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			ui.Column(c.Key("bot-duplicate-confirmation")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewDuplicate2)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				ui.Text(c, botDuplicateReviewText(m.HostLabel, r.Command)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, i18n.T(i18n.KeyW4AssistantTheHostKeepsTheConfigurationAboveInclu)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, i18n.T(i18n.KeyW4AssistantANewIdSessionIdNameAndTimestampsPinn)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botDuplicateAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				hasGoal := pluginrpc.DshbotBotDuplicateHasObjective(r.Command)
				if hasGoal {
					ui.Text(c, i18n.T(i18n.KeyW4AssistantTheCopyKeepsTheFullObjectiveAndRound)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
					executionAck := r.ExecutionAcknowledged
					if ui.Checkbox(c, &executionAck, botDuplicateExecutionAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledgeExecution != nil {
						m.OnAcknowledgeExecution(executionAck)
					}
				}
				if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToDuplicateDraft)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, botDuplicateConfirmLabel()).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged || (hasGoal && !r.ExecutionAcknowledged)).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
