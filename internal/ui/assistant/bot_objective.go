package assistant

import (
	"whaleisle/internal/ui/widgets"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

type BotObjectiveModel struct {
	State                           dshbotstore.ObjectiveSnapshot
	HostLabel                       string
	OnRefresh, OnConfirm, OnDismiss func()
	OnReview                        func(string)
}

func botObjectiveActionLabel(action string) string {
	switch action {
	case "start":
		return i18n.T(i18n.KeyW4AssistantStartResumeObjective)
	case "pause":
		return i18n.T(i18n.KeyMiscAssistantPauseObjective)
	case "clear":
		return i18n.T(i18n.KeyMiscAssistantClearCatalogObjective)
	}
	return action
}
func botObjectiveImpact(action string) string {
	switch action {
	case "start":
		return i18n.T(i18n.KeyW4AssistantResolvesOrResumesThisBotSColdSession)
	case "pause":
		return i18n.T(i18n.KeyW4AssistantRequestsAPauseOnlyWhenTheHostAlready)
	case "clear":
		return i18n.T(i18n.KeyW4AssistantClearsThisBotSCatalogObjectiveTextAnd)
	}
	return ""
}
func botObjectiveStatus(c *ui.Context, s pluginrpc.DshbotObjectiveStatus, fg, muted ui.Color) {
	if !s.GoalKnown {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantLiveGoalNotIncludedRefresh)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		return
	}
	if s.Goal == nil {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoVisibleLiveGoalInThisSnapshot)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		return
	}
	g := s.Goal
	ui.Text(c, fmt.Sprintf("Live phase: %s · activation: %s\nRounds started: %d / %d", g.Phase, g.Activation, g.RoundsStarted, g.MaxGoalRounds)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
	if g.Blocker != nil {
		ui.Text(c, "Host blocker: "+g.Blocker.Code+" · "+g.Blocker.Message).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
	}
}
func botObjectiveBlock(c *ui.Context, m BotObjectiveModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("bot-objective")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantBotObjective)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantStartPauseAndClearTheCatalogObjective)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshObjective)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		widgets.Notice(c, "bot_objective:73", "", s.Notice != "", func() string { return s.Notice })
		for _, warning := range s.Warnings {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantHostWarning2)+warning).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshBeforeAnotherAction)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		b := s.Status.Bot
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantCatalogObjectiveDetail), m.HostLabel, b.Name, b.ID, b.SessionID, s.Status.Revision, b.Objective, b.ObjectiveMaxRounds)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		botObjectiveStatus(c, s.Status, fg, muted)
		if s.Status.ActivityError != "" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantActivityWarning)+s.Status.ActivityError).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		ui.Row(c).Gap(7).Children(func() {
			for _, action := range []string{"start", "pause", "clear"} {
				disabled := blocked || !s.Loaded || s.Review != nil || action == "start" && strings.TrimSpace(b.Objective) == ""
				if m.OnReview != nil && ui.Button(c, botObjectiveActionLabel(action)).Disabled(disabled).Clicked() {
					m.OnReview(action)
				}
			}
		})
		if s.Review != nil {
			r := s.Review
			before := r.Command.Expected
			ui.Column(c.Key("bot-objective-review")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReview)+botObjectiveActionLabel(r.Command.Action)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantObjectiveReviewDetail), m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, before.Bot.Objective, before.Bot.ObjectiveMaxRounds)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				botObjectiveStatus(c, before, fg, muted)
				ui.Text(c, botObjectiveImpact(r.Command.Action)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				if r.Command.Action == "clear" {
					ui.Text(c, i18n.T(i18n.KeyW4AssistantClearingOnlyAppliesTheHostCasToTheCa)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				} else {
					ui.Text(c, i18n.T(i18n.KeyW4AssistantTheOriginalStartPauseHasNoAtomicCasO)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshUncertainOutcomesNeverAutoRe)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Row(c).Gap(8).Children(func() {
					if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelObjectiveAction)).Disabled(blocked).Clicked() {
						m.OnDismiss()
					}
					if m.OnConfirm != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmObjectiveAction)).Disabled(blocked || !s.Loaded).Clicked() {
						m.OnConfirm()
					}
				})
			})
		}
	})
}
