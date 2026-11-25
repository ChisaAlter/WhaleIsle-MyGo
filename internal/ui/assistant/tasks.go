package assistant

import (
	"fmt"
	"strings"
	"time"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/errcopy"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

type TaskModel struct {
	State     dshbotstore.TaskControlSnapshot
	OnRefresh func()
	OnSelect  func(string)
	OnReview  func(action, taskID string)
	OnConfirm func()
	OnDismiss func()
}

func taskActionLabel(action string) string {
	switch action {
	case "pause":
		return i18n.T(i18n.KeyMiscAssistantPause)
	case "resume":
		return i18n.T(i18n.KeyMiscAssistantResume)
	case "cancel":
		return i18n.T(i18n.KeyMiscAssistantCancelTask)
	case "retry":
		return i18n.T(i18n.KeyMiscAssistantRetryTask)
	case "stop":
		return i18n.T(i18n.KeyMiscAssistantStopBot)
	}
	return action
}

func taskControlBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.Tasks.OnRefresh == nil {
		return
	}
	tasks := m.Tasks
	s := tasks.State
	blocked := m.Busy || m.PluginRPCBusy || s.Busy || !m.HarnessReady || !m.DshbotEnabled
	block := ui.Column(c.Key("bot-task-controls")).Gap(8)
	block.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantBotTasks)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			if aButton(c, i18n.T(i18n.KeyMiscAssistantRefreshTasks), blocked || s.Scope.BotID == "").Clicked() {
				tasks.OnRefresh()
			}
		})
		ui.Text(c, i18n.T(i18n.KeyW4AssistantShowsOnlyDelegatedTasksThisBotSentOr)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if s.Scope.BotID == "" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantSelectABotFirst)).TextColor(muted)
			return
		}
		if s.Error != nil {
			ui.Text(c, s.Error.SentenceText()).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		widgets.AckLine(c, "tasks:66", s.Notice)
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshBeforeMakingChanges2)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Loaded {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("%s · %s · revision %d", s.BotName, s.Scope.BotID, s.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Grow(1)
				if tasks.OnReview != nil && aDangerOutline(c, taskActionLabel("stop"), blocked).Clicked() {
					tasks.OnReview("stop", "")
				}
			})
		}
		if s.Loaded && len(s.Tasks) == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoDelegationTasksForThisBot)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		filterKey := s.Scope.HostKey + ":" + s.Scope.BotID
		query := ui.Local(block, "task-query:"+filterKey, func() string { return "" })
		status := ui.Local(block, "task-status:"+filterKey, func() string { return "all" })
		aInput(c.Key("task-search"), query, coss.InputOptions{Label: i18n.T(i18n.KeyMiscAssistantSearchTasksParticipantsOrErrors)})
		aSelect(c.Key("task-status"), status, []string{"all", "queued", "delivered", "completed", "failed", "paused", "cancelled"}, i18n.T(i18n.KeyMiscAssistantTaskStatus), false).FillWidth()
		visible := visibleBotTasks(s.Tasks, *query, *status)
		if s.Loaded && len(s.Tasks) > 0 && len(visible) == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingTasks)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		for _, task := range visible {
			task := task
			card := ui.Column(c.Key("bot-task-"+task.ID)).Border(.5, border).Radius(12).Padding(12, 16).Gap(6)
			chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
			card.Children(func() {
				title := task.ID + " · " + task.Status
				if s.SelectedID == task.ID {
					title = "▾ " + title
				}
				if tasks.OnSelect != nil && aChoice(c, title, s.SelectedID == task.ID, blocked).Clicked() {
					tasks.OnSelect(task.ID)
				}
				ui.Text(c, taskParticipant(task.FromName, task.FromID)+" → "+taskParticipant(task.ToName, task.ToID)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				ui.Text(c, clip(task.Task, 180)).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg)
				if s.SelectedID != task.ID {
					return
				}
				ui.Row(c).Gap(6).Children(func() {
					for _, participant := range []struct{ id, label string }{
						{task.FromID, i18n.T(i18n.KeyMiscAssistantOpenSenderConversation)},
						{task.ToID, i18n.T(i18n.KeyMiscAssistantOpenRecipientConversation)},
					} {
						participant := participant
						if m.Contacts.State.Loaded && m.OnDshbotOpenRosterSession != nil {
							for _, item := range m.Contacts.State.Catalog.Items {
								if item.ID == strings.TrimSpace(participant.id) && item.Kind == "bot" && strings.TrimSpace(item.SessionID) != "" {
									if aButton(c, participant.label, blocked || m.Contacts.State.Busy).Clicked() {
										m.OnDshbotOpenRosterSession(item.ID)
									}
									break
								}
							}
						}
					}
				})
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantTask)+task.Task).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				if len(task.Constraints) > 0 {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantConstraints)+strings.Join(task.Constraints, "\n- ")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				if task.SuccessCriteria != "" {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantSuccessCriteria)+task.SuccessCriteria).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				if task.ResultSummary != "" {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantResult)+task.ResultSummary).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				if task.Error != "" {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantError)+errcopy.SentenceText(task.Error)).FontSize(fonttheme.InterfaceSize(12)).TextColor(design.For(c).Destructive).Selectable()
				}
				ui.Text(c, fmt.Sprintf("Attempts: %d · %s", task.Attempts, taskTime(task.UpdatedAt))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if task.ParentTaskID != "" || task.RetryOf != "" {
					ui.Text(c, i18n.T(i18n.KeyMiscAssistantRetryOf)+firstTaskValue(task.ParentTaskID, task.RetryOf)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
				}
				ui.Row(c).Gap(6).Children(func() {
					for _, action := range []string{"pause", "resume", "cancel", "retry"} {
						action := action
						if pluginrpc.DshbotTaskActionAllowed(action, task.Status) && tasks.OnReview != nil {
							var btn ui.Element
							if action == "cancel" {
								btn = aDangerOutline(c, taskActionLabel(action), blocked || !s.Loaded)
							} else {
								btn = aButton(c, taskActionLabel(action), blocked || !s.Loaded)
							}
							if btn.Clicked() {
								tasks.OnReview(action, task.ID)
							}
						}
					}
				})
				for _, event := range task.Events {
					ui.Text(c, taskTime(event.At)+" · "+event.Type+" · "+event.ActorID+"\n"+event.Detail).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
				}
			})
		}
		if s.Review != nil {
			command := s.Review.Command
			ui.Column(c.Key("bot-task-review")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyW4AssistantConfirm)+taskActionLabel(command.Action)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				ui.Text(c, fmt.Sprintf("Bot: %s (%s)\nTask: %s · revision %d", s.BotName, command.BotID, firstTaskValue(command.TaskID, "—"), command.Revision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				caveat := i18n.T(i18n.KeyW4AssistantOnlyTheTaskStateChangesCompletedToolF)
				if command.Action == "retry" {
					caveat = i18n.T(i18n.KeyW4AssistantCreatesANewTaskAttemptAndWakesTheRec)
				}
				if command.Action == "resume" {
					caveat = i18n.T(i18n.KeyW4AssistantRequeuesTheTaskAndWakesTheReceivingBo)
				}
				if command.Action == "stop" {
					caveat = i18n.T(i18n.KeyW4AssistantStopsThisBotSCurrentSessionForAGroup)
				}
				ui.Text(c, caveat).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				if command.Action == "retry" {
					ui.Text(c, i18n.T(i18n.KeyW4AssistantConfirmingAgainAfterAFailureReusesThe)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				ui.Row(c).Gap(8).Children(func() {
					if tasks.OnDismiss != nil && aButton(c, i18n.T(i18n.KeyMiscAssistantBack), s.Busy).Clicked() {
						tasks.OnDismiss()
					}
					if tasks.OnConfirm != nil {
						var confirmBtn ui.Element
						if command.Action == "cancel" || command.Action == "stop" {
							confirmBtn = aDanger(c, i18n.T(i18n.KeyMiscAssistantConfirmAction), blocked || !s.Loaded || command.Revision != s.Revision)
						} else {
							confirmBtn = aPrimary(c, i18n.T(i18n.KeyMiscAssistantConfirmAction), blocked || !s.Loaded || command.Revision != s.Revision)
						}
						if confirmBtn.Clicked() {
							tasks.OnConfirm()
						}
					}
				})
			})
		}
	})
}
func taskParticipant(name, id string) string {
	if name != "" {
		return name + " (" + id + ")"
	}
	return id
}
func firstTaskValue(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func taskTime(at int64) string {
	if at <= 0 {
		return "—"
	}
	return time.UnixMilli(at).Local().Format("2006-01-02 15:04:05")
}
