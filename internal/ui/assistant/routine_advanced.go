package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"slices"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func routineWatchAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandCommandEffectsAndWorking)
}

func routineContextName(id string, rows []pluginrpc.DshbotRoutine) string {
	if id == "self" {
		return i18n.T(i18n.KeyMiscAssistantThisRoutineSPreviousOutputSelf)
	}
	for _, r := range rows {
		if r.ID == id {
			return r.Name + " (" + id + ")"
		}
	}
	return i18n.T(i18n.KeyMiscAssistantUnavailableSourceRemoveExplicitlyBef) + id
}
func routineAdvancedEditor(c *ui.Context, m RoutineModel, disabled bool, fg, muted, accent ui.Color) {
	routineAdvancedFields(c, m.State.Draft.Advanced, m.State.Scope.BotID, m.State.SelectedID, m.State.Routines, m.OnChange, disabled, fg, muted, accent)
}
func routineAdvancedFields(c *ui.Context, d dshbotstore.RoutineAdvancedDraft, botID, currentID string, rows []pluginrpc.DshbotRoutine, onChange func(string, string), disabled bool, fg, muted, accent ui.Color) {
	ui.Column(c.Key("routine-advanced")).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantAdvancedSettings)).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantOnlyExplicitlyChangedAdvancedFieldsAre)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		choices := []string{"self"}
		for _, r := range rows {
			if r.BotID == botID && r.ID != currentID {
				choices = append(choices, r.ID)
			}
		}
		for _, id := range d.ContextFrom {
			if !slices.Contains(choices, id) {
				choices = append(choices, id)
			}
		}
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantContextSources)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		for _, id := range choices {
			id := id
			checked := slices.Contains(d.ContextFrom, id)
			if ui.Checkbox(c.Key("routine-context-"+id), &checked, routineContextName(id, rows)).Disabled(disabled || !checked && len(d.ContextFrom) >= 3).Changed() && onChange != nil {
				onChange("contextToggle", id)
			}
		}
		silent := d.SilentAllowed
		if ui.Checkbox(c, &silent, i18n.T(i18n.KeyMiscAssistantAllowSilentResponse)).Disabled(disabled).Changed() && onChange != nil {
			onChange("silentAllowed", fmt.Sprint(silent))
		}
		ui.Text(c, i18n.T(i18n.KeyW4AssistantSilenceOnlyChangesHowNormalOutputIsDe)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if d.WatchUnknown {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantUnknownWatchPreserved)+d.WatchRaw).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent).Selectable()
		}
		ui.Row(c).Gap(8).Children(func() {
			for _, choice := range []struct{ id, label string }{{"", i18n.T(i18n.KeyMiscAssistantNoWatch)}, {"url", i18n.T(i18n.KeyMiscAssistantUrlWatch)}, {"command", i18n.T(i18n.KeyMiscAssistantCommandWatch)}} {
				choice := choice
				if ui.Button(c, choice.label).Disabled(disabled || !d.WatchUnknown && d.WatchKind == choice.id).Clicked() && onChange != nil {
					onChange("watchKind", choice.id)
				}
			}
		})
		if !d.WatchUnknown && d.WatchKind != "" {
			value, timeout := d.WatchValue, d.WatchTimeout
			label := i18n.T(i18n.KeyMiscAssistantFullWatchUrl)
			if d.WatchKind == "command" {
				label = i18n.T(i18n.KeyMiscAssistantFullHostShellCommand)
			}
			ui.TextArea(c.Key("routine-watch-value"), &value).Label(label).Lines(2, 6).FillWidth().Disabled(disabled).OnChange(func() {
				if onChange != nil {
					onChange("watchValue", value)
				}
			})
			ui.TextInput(c.Key("routine-watch-timeout"), &timeout).Label(i18n.T(i18n.KeyMiscAssistantTimeoutMs1300000)).FillWidth().Disabled(disabled).OnChange(func() {
				if onChange != nil {
					onChange("watchTimeout", timeout)
				}
			})
			ui.Text(c, i18n.T(i18n.KeyW4AssistantAnUnchangedCheckMaySkipTheRunAFailed)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
		}
	})
}
func routineAdvancedReview(c *ui.Context, m RoutineModel, r dshbotstore.RoutineReview, fg, muted, accent ui.Color) {
	d := r.Advanced
	sources := []string{}
	for _, id := range d.ContextFrom {
		sources = append(sources, routineContextName(id, m.State.Routines))
	}
	if len(sources) == 0 {
		sources = []string{i18n.T(i18n.KeyMiscAssistantNone)}
	}
	ui.Text(c, i18n.T(i18n.KeyMiscAssistantEffectiveContextSources)+strings.Join(sources, "\n")+fmt.Sprintf("\nSilent allowed: %t", d.SilentAllowed)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
	ui.Text(c, i18n.T(i18n.KeyW4AssistantLaterRunsFeedThePreviousOutputOfTheS)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if d.WatchUnknown {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantEffectiveWatchUnrecognized)+d.WatchRaw).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
	} else if d.WatchKind == "" {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantEffectiveWatchNone)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	} else {
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantEffectiveWatch), d.WatchKind, d.WatchValue, d.WatchTimeout)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if d.WatchKind == "url" {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantBeforeEachAutomaticRunTheHostReadsThi)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
		}
	}
	if d.WatchKind == "command" || d.WatchUnknown {
		workspace := r.WorkspaceID
		if workspace == "" {
			workspace = i18n.T(i18n.KeyMiscAssistantNotConfigured)
		}
		ui.Text(c, i18n.T(i18n.KeyW4AssistantConfiguredWorkspaceidRealCwdUnverified)+workspace+i18n.T(i18n.KeyW4AssistantNruntimeWorkingDirectoryUseThatPathWhe)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
	}
	if d.WatchKind != "" || d.WatchUnknown {
		ui.Text(c, i18n.T(i18n.KeyW4AssistantAFailedWatchCheckDuringAnAutomaticRun)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	}
	if dshbotstore.RoutineReviewNeedsWatchAcknowledgement(r) {
		ack := r.WatchAcknowledged
		if ui.Checkbox(c, &ack, routineWatchAcknowledgement()).Disabled(m.State.Busy).Changed() && m.OnAcknowledgeWatch != nil {
			m.OnAcknowledgeWatch(ack)
		}
	}
}
