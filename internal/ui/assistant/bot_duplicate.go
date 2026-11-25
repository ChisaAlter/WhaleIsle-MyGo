package assistant

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
		if m.OnRefresh != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantReadDuplicateSource), blocked || s.Scope == "" || m.SourceID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != nil {
			ui.Text(c, s.Error.SentenceText()).TextColor(accent).Selectable()
		}
		widgets.AckLine(c, "bot_duplicate:75", s.Notice)
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
			if m.OnOpen != nil && aPrimary(c, i18n.T(i18n.KeyMiscAssistantOpenReceiptLinkedCopy), blocked).Clicked() {
				m.OnOpen(s.Created.ID)
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && aPrimary(c, i18n.T(i18n.KeyMiscAssistantPrepareDuplicate), blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		name := d.Name
		aInput(c.Key("bot-duplicate-name"), &name, coss.InputOptions{Label: i18n.T(i18n.KeyMiscAssistantNewStableName120), Disabled: disabled}).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("name", name)
			}
		})
		cwd := d.ScratchCwd
		aInput(c.Key("bot-duplicate-cwd"), &cwd, coss.InputOptions{Label: i18n.T(i18n.KeyMiscAssistantScratchCwdOverridden), Disabled: disabled}).OnChange(func() {
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
				if m.OnOpen != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantOpenInspectedCandidate)+row.ID, blocked).Clicked() {
					m.OnOpen(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantDiscardDuplicateDraft), blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && aPrimary(c, i18n.T(i18n.KeyMiscAssistantReviewDuplicate), disabled || !s.Loaded || !s.Writable).Clicked() {
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
				if aCheckbox(c, &ack, botDuplicateAcknowledgement(), blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				hasGoal := pluginrpc.DshbotBotDuplicateHasObjective(r.Command)
				if hasGoal {
					ui.Text(c, i18n.T(i18n.KeyW4AssistantTheCopyKeepsTheFullObjectiveAndRound)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
					executionAck := r.ExecutionAcknowledged
					if aCheckbox(c, &executionAck, botDuplicateExecutionAcknowledgement(), blocked).Changed() && m.OnAcknowledgeExecution != nil {
						m.OnAcknowledgeExecution(executionAck)
					}
				}
				if m.OnDismiss != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantBackToDuplicateDraft), blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && aPrimary(c, botDuplicateConfirmLabel(), blocked || !s.Loaded || !s.Writable || !r.Acknowledged || (hasGoal && !r.ExecutionAcknowledged)).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
