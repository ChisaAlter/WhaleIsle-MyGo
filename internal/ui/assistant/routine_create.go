package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

type RoutineCreateModel struct {
	State                                                                   dshbotstore.RoutineCreateSnapshot
	HostLabel, BotName                                                      string
	Routines                                                                []pluginrpc.DshbotRoutine
	OnAcknowledgeWatch                                                      func(bool)
	OnStart, OnReview, OnConfirm, OnDismiss, OnDiscard, OnRebase, OnRefresh func()
	OnChange                                                                func(field, value string)
	OnAdopt                                                                 func(id string)
}

func routineCreateImpact(enabled bool) string {
	if enabled {
		return "将创建并启用例行任务。Host 会计算下次时间并自动调度，Bot 可能执行工具、发送外部消息并产生模型费用；本次保存不会额外调用立即运行。"
	}
	return "将创建暂停的例行任务，后续须单独确认启用才会自动调度。本次保存不会调用立即运行；启用后可能执行工具、发送外部消息并产生模型费用。"
}
func routineCreateBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	w := m.RoutineCreate
	s := w.State
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || w.OnStart == nil {
		return
	}
	blocked := m.Busy || m.PluginRPCBusy || s.Busy || !m.HarnessReady || !m.DshbotEnabled
	ui.Column(c.Key("routine-create")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCreateBotRoutine)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if s.Draft == nil {
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantNewRoutine)).Disabled(blocked || !s.Loaded || s.Scope.BotID == "").Clicked() {
				w.OnStart()
			}
			if !s.Loaded {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantRefreshTheRoutineCatalogFirstEmpty)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantSaveScheduleContextSilentRepliesAnd)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		frequency := d.Frequency
		if frequency == "" {
			frequency = "daily"
		}
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantFrequency)+routineCreateFrequencyLabel(frequency)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		for _, choices := range [][]string{{"once", "hourly", "daily"}, {"weekdays", "weekly", "monthly"}, {"interval", "advanced"}} {
			choices := choices
			ui.Row(c).Gap(8).Children(func() {
				for _, choice := range choices {
					choice := choice
					if ui.Button(c, routineCreateFrequencyLabel(choice)).Disabled(disabled || frequency == choice).Clicked() && w.OnChange != nil {
						w.OnChange("frequency", choice)
					}
				}
			})
		}
		ui.Text(c, routineCreateTimingHint(frequency)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · revision %d", w.HostLabel, w.BotName, s.Scope.BotID, d.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		fields := []struct{ id, label, value string }{{"name", i18n.T(i18n.KeyMiscAssistantNewRoutineName), d.Name}}
		switch frequency {
		case "hourly":
			// Original hourly is exactly every 1h and has no clock-time fields.
		case "advanced":
			fields = append(fields, struct{ id, label, value string }{"cron", i18n.T(i18n.KeyMiscAssistantMinuteHourDayMonthWeekday), d.Cron})
		case "once":
			fields = append(fields, struct{ id, label, value string }{"onceDelay", i18n.T(i18n.KeyMiscAssistantDelayLabel), d.OnceDelay})
		case "interval":
			fields = append(fields, struct{ id, label, value string }{"intervalAmount", i18n.T(i18n.KeyMiscAssistantIntervalAmountPositiveInteger), d.IntervalAmount})
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantIntervalUnit)+d.IntervalUnit).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Row(c).Gap(8).Children(func() {
				for _, choice := range []struct{ id, label string }{{"m", i18n.T(i18n.KeyMiscAssistantMinutes)}, {"h", i18n.T(i18n.KeyMiscAssistantHours)}, {"d", i18n.T(i18n.KeyMiscAssistantDays)}} {
					choice := choice
					if ui.Button(c, choice.label).Disabled(disabled || d.IntervalUnit == choice.id).Clicked() && w.OnChange != nil {
						w.OnChange("intervalUnit", choice.id)
					}
				}
			})
		default:
			fields = append(fields, struct{ id, label, value string }{"hour", i18n.T(i18n.KeyMiscAssistantHour0023), d.Hour}, struct{ id, label, value string }{"minute", i18n.T(i18n.KeyMiscAssistantMinute0059), d.Minute})
		}
		if frequency == "weekly" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWeekday)+d.Weekday+i18n.T(i18n.KeyMiscAssistantSunday)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			for _, choices := range [][]struct{ id, label string }{{{"0", i18n.T(i18n.KeyMiscAssistantSun)}, {"1", i18n.T(i18n.KeyMiscAssistantMon)}, {"2", i18n.T(i18n.KeyMiscAssistantTue)}, {"3", i18n.T(i18n.KeyMiscAssistantWed)}}, {{"4", i18n.T(i18n.KeyMiscAssistantThu)}, {"5", i18n.T(i18n.KeyMiscAssistantFri)}, {"6", i18n.T(i18n.KeyMiscAssistantSat)}}} {
				choices := choices
				ui.Row(c).Gap(8).Children(func() {
					for _, choice := range choices {
						choice := choice
						if ui.Button(c, choice.label).Disabled(disabled || d.Weekday == choice.id).Clicked() && w.OnChange != nil {
							w.OnChange("weekday", choice.id)
						}
					}
				})
			}
		}
		if frequency == "monthly" {
			fields = append(fields, struct{ id, label, value string }{"monthDay", i18n.T(i18n.KeyMiscAssistantDayOfMonth131), d.MonthDay})
		}
		fields = append(fields, struct{ id, label, value string }{"timezone", i18n.T(i18n.KeyMiscAssistantTimezoneLabel), d.Timezone}, struct{ id, label, value string }{"maxRuns", i18n.T(i18n.KeyMiscAssistantMaxRuns0Unlimited), d.MaxRuns})
		for _, f := range fields {
			f := f
			value := f.value
			ui.TextInput(c.Key("create-"+f.id), &value).Label(f.label).FillWidth().Disabled(disabled).OnChange(func() {
				if w.OnChange != nil {
					w.OnChange(f.id, value)
				}
			})
		}
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantSpecifyTheIntendedIanaTimezone)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		prompt := d.Prompt
		ui.TextArea(c.Key("create-prompt"), &prompt).Label(i18n.T(i18n.KeyMiscAssistantNewRoutinePrompt)).Lines(4, 10).FillWidth().Disabled(disabled).OnChange(func() {
			if w.OnChange != nil {
				w.OnChange("prompt", prompt)
			}
		})
		label := i18n.T(i18n.KeyMiscAssistantNewRoutinePaused)
		if d.Enabled {
			label = i18n.T(i18n.KeyMiscAssistantNewRoutineEnabled)
		}
		if w.OnChange != nil && ui.Button(c, label).Disabled(disabled).Clicked() {
			w.OnChange("enabled", fmt.Sprint(!d.Enabled))
		}
		routineAdvancedFields(c.Key("create-advanced"), d.Advanced, s.Scope.BotID, "", w.Routines, w.OnChange, disabled, fg, muted, accent)
		if s.Uncertain {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantThisCreationIsLockedUntilItsOutcom)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if w.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantCheckCreationOutcome)).Disabled(blocked).Clicked() {
				w.OnRefresh()
			}
			for _, row := range s.Candidates {
				row := row
				ui.Column(c.Key("create-candidate-" + row.ID)).Background(field).Padding(10).Gap(6).Children(func() {
					ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantNewRecord), row.Name, row.ID, row.Prompt, row.Schedule, row.Timezone, row.Enabled)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
					if w.OnAdopt != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantUseRecord)+row.ID).Disabled(blocked).Clicked() {
						w.OnAdopt(row.ID)
					}
				})
			}
			if s.RetryVerified && w.OnRebase != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantNoNewRecordReviewAgain)).Disabled(blocked).Clicked() {
				w.OnRebase()
			}
		} else {
			ui.Row(c).Gap(8).Children(func() {
				if w.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardNewDraft)).Disabled(blocked).Clicked() {
					w.OnDiscard()
				}
				if w.OnRebase != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRecheckCatalog)).Disabled(blocked || !s.Loaded).Clicked() {
					w.OnRebase()
				}
				if w.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewCreation)).Disabled(disabled || !s.Loaded).Clicked() {
					w.OnReview()
				}
			})
		}
		if s.Review != nil {
			cmd := s.Review.Command
			ui.Column(c.Key("create-review")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewCreation2)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s\nName: %s · revision %d\nPrompt: %s\nSchedule: %s · %s\nEnabled: %t · Max runs: %s", w.HostLabel, w.BotName, cmd.BotID, cmd.SessionID, cmd.Edit.Name, cmd.Revision, cmd.Edit.Prompt, cmd.Edit.Schedule, cmd.Edit.Timezone, cmd.Edit.Enabled, routineLimit(cmd.Edit.MaxRuns))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantFrequency)+routineCreateFrequencyLabel(frequency)+"\n"+routineCreateTimingHint(frequency)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				ui.Text(c, routineCreateImpact(cmd.Edit.Enabled)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				routineAdvancedReview(c, RoutineModel{State: dshbotstore.RoutineSnapshot{Routines: w.Routines, Busy: s.Busy}, OnAcknowledgeWatch: w.OnAcknowledgeWatch}, dshbotstore.RoutineReview{Command: pluginrpc.DshbotRoutineCommand{Action: "save", Edit: cmd.Edit, Advanced: cmd.Advanced}, Advanced: s.Review.Advanced, WorkspaceID: s.Review.WorkspaceID, WatchAcknowledged: s.Review.WatchAcknowledged}, fg, muted, accent)
				ui.Row(c).Gap(8).Children(func() {
					if w.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToDraft2)).Disabled(s.Busy).Clicked() {
						w.OnDismiss()
					}
					if w.OnConfirm != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmNewRoutine)).Disabled(blocked || !s.Loaded || dshbotstore.RoutineCreateNeedsWatchAcknowledgement(*s.Review) && !s.Review.WatchAcknowledged).Clicked() {
						w.OnConfirm()
					}
				})
			})
		}
	})
}
