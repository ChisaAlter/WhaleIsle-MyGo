package assistant

import (
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

type RoutineModel struct {
	Notify                                             BotNotifyModel
	Avatar                                             BotAvatarModel
	SenderPolicy                                       BotSenderPolicyModel
	TextProfile                                        BotTextProfileModel
	Route                                              BotRouteModel
	ObjectiveConfig                                    BotObjectiveConfigModel
	MemoryReview                                       BotMemoryReviewModel
	Capabilities                                       BotCapabilityModel
	Objective                                          BotObjectiveModel
	Memory                                             BotMemoryModel
	State                                              dshbotstore.RoutineSnapshot
	HostLabel                                          string
	OnRefresh, OnEdit, OnDiscard, OnConfirm, OnDismiss func()
	OnSelect                                           func(string)
	OnChange                                           func(field, value string)
	OnReview                                           func(action string)
	OnAcknowledgeWatch                                 func(bool)
	OnPrepareTrigger, OnConfirmTrigger, OnCloseTrigger func()
	OnAcknowledgeTrigger                               func(bool)
	OnCopyTrigger                                      func(string)
	TriggerMaterial                                    func() (string, string)
	TriggerUnavailable                                 string
}

func routineActionLabel(action string, enabled bool) string {
	switch action {
	case "save":
		return i18n.T(i18n.KeyMiscAssistantReviewSave)
	case "toggle":
		if enabled {
			return i18n.T(i18n.KeyMiscAssistantPauseRoutine)
		}
		return i18n.T(i18n.KeyMiscAssistantEnableRoutine)
	case "run":
		return i18n.T(i18n.KeyMiscAssistantRunNow)
	case "delete":
		return i18n.T(i18n.KeyMiscAssistantDeleteRoutine)
	}
	return action
}
func routineImpact(cmd pluginrpc.DshbotRoutineCommand) string {
	switch cmd.Action {
	case "delete":
		return i18n.T(i18n.KeyW4AssistantDeletesThisRoutineAndItsDedicatedNotep)
	case "run":
		return i18n.T(i18n.KeyW4AssistantTriggersThisBotImmediatelyItMayRunToo)
	case "toggle":
		if cmd.EnabledBefore {
			return i18n.T(i18n.KeyW4AssistantFutureSchedulingPausesAndUndeliveredTas)
		}
		return i18n.T(i18n.KeyW4AssistantEnablesFutureSchedulingAndRecomputesThe)
	case "save":
		if cmd.Edit.Enabled {
			return i18n.T(i18n.KeyW4AssistantSavesTheConfigurationBelowAndRecomputes)
		}
		return i18n.T(i18n.KeyW4AssistantSavesTheConfigurationBelowWhileStaying)
	}
	return ""
}
func routineControlBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.Routines.OnRefresh == nil {
		return
	}
	r := m.Routines
	s := r.State
	blocked := m.Busy || m.PluginRPCBusy || s.Busy || !m.HarnessReady || !m.DshbotEnabled
	block := ui.Column(c.Key("bot-routines")).Gap(8)
	block.Children(func() {
		botTextProfileBlock(c, r.TextProfile, blocked, fg, muted, accent)
		botSenderPolicyBlock(c, r.SenderPolicy, blocked, fg, muted, accent)
		botNotifyBlock(c, r.Notify, blocked, fg, muted, accent)
		botAvatarBlock(c, r.Avatar, blocked, fg, muted, accent)
		botRouteBlock(c, r.Route, blocked, fg, muted, accent)
		botMemoryBlock(c, r.Memory, blocked, fg, muted, accent)
		botMemoryReviewBlock(c, r.MemoryReview, blocked, fg, muted, accent)
		botCapabilitiesBlock(c, r.Capabilities, blocked, fg, muted, accent)
		botObjectiveBlock(c, r.Objective, blocked, fg, muted, accent)
		botObjectiveConfigBlock(c, r.ObjectiveConfig, blocked, fg, muted, accent)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantBotRoutines)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshRoutines)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
				r.OnRefresh()
			}
		})
		ui.Text(c, i18n.T(i18n.KeyW4AssistantReadsTheSelectedBotSRealRoutinesBelow)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if s.Scope.BotID == "" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantSelectABotFirst)).TextColor(muted)
			return
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshBeforeMakingChanges)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Busy {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWaitingForHost)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Loaded {
			ui.Text(c, fmt.Sprintf("%s · Bot %s · revision %d", s.BotName, s.Scope.BotID, s.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if len(s.Routines) == 0 {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoRoutinesForThisBot)).TextColor(muted)
			}
		}
		// Original dismissals last for this canonical session, without changing
		// the Host's missed-run records or any scheduling configuration.
		dismissed := ui.Local(block, "routine-missed:"+s.Scope.HostKey+":"+s.SessionID, func() map[string]bool { return map[string]bool{} })
		missed := []string{}
		if s.MissedKnown {
			for _, id := range s.MissedIDs {
				if !(*dismissed)[id] {
					missed = append(missed, id)
				}
			}
		}
		if len(missed) > 0 {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantMissedRuns), len(missed))).TextColor(accent)
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDismissMissedRunNotice)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
				for _, id := range missed {
					(*dismissed)[id] = true
				}
			}
		}
		query := ui.Local(block, "routine-query:"+s.Scope.HostKey+":"+s.Scope.BotID, func() string { return "" })
		ui.TextInput(c.Key("routine-search"), query).Label(i18n.T(i18n.KeyMiscAssistantSearchRoutinesBotsOrErrors)).FillWidth().Height(32).Radius(8).Border(.5, border)
		visible := visibleBotRoutines(s.Routines, s.BotName, *query)
		if s.Loaded && len(s.Routines) > 0 && len(visible) == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingRoutines)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		for _, row := range visible {
			row := row
			card := ui.Column(c.Key("bot-routine-"+row.ID)).Border(.5, border).Radius(12).Padding(12, 16).Gap(6)
			chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
			card.Children(func() {
				status := i18n.T(i18n.KeyMiscAssistantPaused)
				if row.Enabled {
					status = i18n.T(i18n.KeyMiscAssistantEnabled3)
				}
				title := firstTaskValue(row.Name, row.ID) + " · " + status
				if r.OnSelect != nil && ui.Button(c, title).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
					r.OnSelect(row.ID)
				}
				ui.Text(c, fmt.Sprintf("%s · %s · %s", row.ID, row.Schedule, row.Timezone)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				for _, id := range s.MissedIDs {
					if id == row.ID && s.MissedKnown && !(*dismissed)[id] {
						ui.Text(c, i18n.T(i18n.KeyMiscAssistantMissedScheduledRun)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
					}
				}
				if s.SelectedID != row.ID {
					return
				}
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantPrompt)+row.Prompt).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, fmt.Sprintf("Next: %s · Last: %s\nRuns: %d / %s · Outcome: %s", taskTime(row.NextRunAt), taskTime(row.LastRunAt), row.RunCount, routineLimit(row.MaxRuns), firstTaskValue(row.LastOutcome, "—"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
				if row.LastError != "" {
					ui.Text(c, row.LastError).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				if row.LastOutput != "" {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantLastOutput)+row.LastOutput).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				ui.Text(c, fmt.Sprintf("Context: %s · Silent allowed: %t\nWatch: %s", strings.Join(row.ContextFrom, ", "), row.SilentAllowed, firstTaskValue(string(row.Watch), "—"))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
				routineTriggerBlock(c, r, blocked, fg, muted, accent)
				ui.Row(c).Gap(6).Children(func() {
					if r.OnEdit != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantEditRoutine)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded).Clicked() {
						r.OnEdit()
					}
					for _, action := range []string{"toggle", "run", "delete"} {
						action := action
						if r.OnReview != nil && ui.Button(c, routineActionLabel(action, row.Enabled)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || s.Review != nil).Clicked() {
							r.OnReview(action)
						}
					}
				})
				if s.Draft != nil && s.Draft.ID == row.ID {
					routineEditor(c, r, blocked, fg, muted, border, accent)
				}
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantRunHistoryLatest10)).FontSize(fonttheme.InterfaceSize(12)).FontWeight(600).TextColor(fg)
				if row.RunHistory == nil {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantHistoryUnavailable)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				} else if len(row.RunHistory) == 0 {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoRecordedRuns)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				for i := len(row.RunHistory) - 1; i >= 0 && i >= len(row.RunHistory)-10; i-- {
					run := row.RunHistory[i]
					ui.Text(c, fmt.Sprintf("%s · %s · %s\nCreated: %s · Started: %s · Ended: %s\nSession: %s · Turn: %d", run.RunID, run.Status, run.Trigger, taskTime(run.CreatedAt), taskTime(run.StartedAt), taskTime(run.EndedAt), run.SessionID, run.Turn)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
					if run.Error != "" {
						ui.Text(c, run.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent).Selectable()
					}
				}
			})
		}
		if s.Review != nil {
			review := s.Review
			cmd := review.Command
			ui.Column(c.Key("routine-review")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReview)+routineActionLabel(cmd.Action, cmd.EnabledBefore)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s\nRoutine: %s (%s) · revision %d", r.HostLabel, s.BotName, cmd.BotID, review.SessionID, review.Name, cmd.ID, cmd.Revision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "Prompt: "+review.Prompt+"\nSchedule: "+review.Schedule+" · "+review.Timezone).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				if cmd.Action == "save" {
					ui.Text(c, fmt.Sprintf("Enabled: %t · Max runs: %s", cmd.Edit.Enabled, routineLimit(cmd.Edit.MaxRuns))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				}
				ui.Text(c, routineImpact(cmd)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				if cmd.Action != "delete" {
					routineAdvancedReview(c, r, *review, fg, muted, accent)
				}
				ui.Row(c).Gap(8).Children(func() {
					if r.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToRoutine)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(s.Busy).Clicked() {
						r.OnDismiss()
					}
					if r.OnConfirm != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmRoutineAction)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Loaded || cmd.Revision != s.Revision || dshbotstore.RoutineReviewNeedsWatchAcknowledgement(*review) && !review.WatchAcknowledged).Clicked() {
						r.OnConfirm()
					}
				})
			})
		}
	})
}
func routineLimit(n int) string {
	if n == 0 {
		return i18n.T(i18n.KeyMiscAssistantUnlimited)
	}
	return fmt.Sprint(n)
}
func routineEditor(c *ui.Context, m RoutineModel, blocked bool, fg, muted, border, accent ui.Color) {
	d := *m.State.Draft
	disabled := blocked || m.State.Review != nil
	ui.Column(c.Key("routine-editor-" + d.ID)).Gap(7).Children(func() {
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantLocalDraft), d.Revision)).FontSize(fonttheme.InterfaceSize(12)).FontWeight(600).TextColor(fg)
		if d.Revision != m.State.Revision {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantDraftRevisionIsStale)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		}
		for _, f := range []struct{ id, label, value string }{{"name", i18n.T(i18n.KeyMiscAssistantRoutineName), d.Edit.Name}, {"schedule", i18n.T(i18n.KeyMiscAssistantSchedule), d.Edit.Schedule}, {"timezone", i18n.T(i18n.KeyMiscAssistantTimezone), d.Edit.Timezone}, {"maxRuns", i18n.T(i18n.KeyMiscAssistantMaxRuns0Unlimited), d.MaxRunsText}} {
			f := f
			value := f.value
			ui.TextInput(c.Key("routine-field-"+f.id), &value).Label(f.label).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, value)
				}
			})
		}
		ui.Text(c, i18n.T(i18n.KeyW4AssistantRuleExamples30mOnceEvery1h0915Vali)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		prompt := d.Edit.Prompt
		ui.TextArea(c.Key("routine-prompt"), &prompt).Label(i18n.T(i18n.KeyMiscAssistantRoutinePrompt)).Lines(4, 10).Radius(8).Border(.5, border).FillWidth().Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("prompt", prompt)
			}
		})
		routineAdvancedEditor(c, m, disabled, fg, muted, accent)
		enabledLabel := i18n.T(i18n.KeyMiscAssistantDraftPaused)
		if d.Edit.Enabled {
			enabledLabel = i18n.T(i18n.KeyMiscAssistantDraftEnabled)
		}
		if m.OnChange != nil && ui.Button(c, enabledLabel).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(disabled).Clicked() {
			m.OnChange("enabled", fmt.Sprint(!d.Edit.Enabled))
		}
		ui.Row(c).Gap(8).Children(func() {
			if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardRoutineDraft)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, routineActionLabel("save", d.Edit.Enabled)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(disabled || !m.State.Loaded || d.Revision != m.State.Revision).Clicked() {
				m.OnReview("save")
			}
		})
	})
}
