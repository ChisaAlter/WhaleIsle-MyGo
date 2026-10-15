package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
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
		return "核对并保存… / Review save…"
	case "toggle":
		if enabled {
			return "暂停例行任务… / Pause routine…"
		}
		return "启用例行任务… / Enable routine…"
	case "run":
		return "立即运行… / Run now…"
	case "delete":
		return "删除例行任务… / Delete routine…"
	}
	return action
}
func routineImpact(cmd pluginrpc.DshbotRoutineCommand) string {
	switch cmd.Action {
	case "delete":
		return "将删除此例行任务及其专属 notepad，并取消尚未投递的任务邮件。不会中止正在运行的模型，也不会撤销已发生的文件、工具或外部操作。"
	case "run":
		return "将立即触发此 Bot，可能执行工具、发送外部消息并产生模型费用。暂停状态也可手动触发；返回目录不代表执行成功。结果不确定时先刷新，不要重复运行。"
	case "toggle":
		if cmd.EnabledBefore {
			return "将暂停后续调度，并取消尚未投递的任务邮件。不会中止正在运行的模型，也不会撤销已发生的操作。"
		}
		return "将启用后续调度并重算下次时间，Bot 可能按提示词执行工具、发送外部消息并产生模型费用。"
	case "save":
		if cmd.Edit.Enabled {
			return "将保存以下配置并重算下次执行时间，按启用配置调度。Bot 可能按此提示词周期执行工具、发送外部消息并产生模型费用。仅明确修改的高级字段会一并提交，未改字段由 Host 保留。"
		}
		return "将保存以下配置并保持暂停，重算下次时间、取消未投递邮件。不会停止已运行的模型；仅明确修改的高级字段会一并提交，未改字段由 Host 保留。"
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
	ui.Column(c.Key("bot-routines")).Gap(8).Children(func() {
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
			ui.Text(c, "Bot 例行任务 / Bot routines").FontSize(13).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, "刷新例行任务 / Refresh routines").Disabled(blocked || s.Scope.BotID == "").Clicked() {
				r.OnRefresh()
			}
		})
		ui.Text(c, "读取所选 Bot 的真实例行任务。下方可新建八种时间类型；已有任务可编辑上下文、静默及 watch，并可独立审阅获取当前 Host 目录共享触发凭据。").FontSize(11).TextColor(muted)
		if s.Scope.BotID == "" {
			ui.Text(c, "先选择当前 Host 的 Bot / Select a Bot first").TextColor(muted)
			return
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(12).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "请刷新读取当前 Host。未确认结果前不开放写操作 / Refresh before making changes.").FontSize(11).TextColor(muted)
		}
		if s.Busy {
			ui.Text(c, "等待 Host… / Waiting for Host…").FontSize(11).TextColor(muted)
		}
		if s.Loaded {
			ui.Text(c, fmt.Sprintf("%s · Bot %s · revision %d", s.BotName, s.Scope.BotID, s.Revision)).FontSize(11).TextColor(muted)
			if len(s.Routines) == 0 {
				ui.Text(c, "此 Bot 暂无例行任务 / No routines for this Bot").TextColor(muted)
			}
		}
		for _, row := range s.Routines {
			row := row
			ui.Column(c.Key("bot-routine-"+row.ID)).Background(field).Border(1, border).Radius(8).Padding(10).Gap(6).Children(func() {
				status := "暂停 / Paused"
				if row.Enabled {
					status = "启用 / Enabled"
				}
				title := firstTaskValue(row.Name, row.ID) + " · " + status
				if r.OnSelect != nil && ui.Button(c, title).Disabled(blocked).Clicked() {
					r.OnSelect(row.ID)
				}
				ui.Text(c, fmt.Sprintf("%s · %s · %s", row.ID, row.Schedule, row.Timezone)).FontSize(11).TextColor(muted)
				for _, id := range s.MissedIDs {
					if id == row.ID && s.MissedKnown {
						ui.Text(c, "Host 标记为错过执行 / Missed scheduled run").FontSize(11).TextColor(accent)
					}
				}
				if s.SelectedID != row.ID {
					return
				}
				ui.Text(c, "提示词 / Prompt: "+row.Prompt).FontSize(12).TextColor(fg).Selectable()
				ui.Text(c, fmt.Sprintf("Next: %s · Last: %s\nRuns: %d / %s · Outcome: %s", taskTime(row.NextRunAt), taskTime(row.LastRunAt), row.RunCount, routineLimit(row.MaxRuns), firstTaskValue(row.LastOutcome, "—"))).FontSize(11).TextColor(muted).Selectable()
				if row.LastError != "" {
					ui.Text(c, row.LastError).FontSize(12).TextColor(accent).Selectable()
				}
				if row.LastOutput != "" {
					ui.Text(c, "上次输出 / Last output: "+row.LastOutput).FontSize(12).TextColor(fg).Selectable()
				}
				ui.Text(c, fmt.Sprintf("Context: %s · Silent allowed: %t\nWatch: %s", strings.Join(row.ContextFrom, ", "), row.SilentAllowed, firstTaskValue(string(row.Watch), "—"))).FontSize(11).TextColor(muted).Selectable()
				routineTriggerBlock(c, r, blocked, fg, muted, accent)
				ui.Row(c).Gap(6).Children(func() {
					if r.OnEdit != nil && ui.Button(c, "编辑例行任务 / Edit routine").Disabled(blocked || !s.Loaded).Clicked() {
						r.OnEdit()
					}
					for _, action := range []string{"toggle", "run", "delete"} {
						action := action
						if r.OnReview != nil && ui.Button(c, routineActionLabel(action, row.Enabled)).Disabled(blocked || !s.Loaded || s.Review != nil).Clicked() {
							r.OnReview(action)
						}
					}
				})
				if s.Draft != nil && s.Draft.ID == row.ID {
					routineEditor(c, r, blocked, fg, muted, accent)
				}
				ui.Text(c, "运行历史 / Run history (latest 10)").FontSize(12).Bold().TextColor(fg)
				if row.RunHistory == nil {
					ui.Text(c, "Host 未提供运行历史 / History unavailable").FontSize(11).TextColor(muted)
				} else if len(row.RunHistory) == 0 {
					ui.Text(c, "暂无运行记录 / No recorded runs").FontSize(11).TextColor(muted)
				}
				for i := len(row.RunHistory) - 1; i >= 0 && i >= len(row.RunHistory)-10; i-- {
					run := row.RunHistory[i]
					ui.Text(c, fmt.Sprintf("%s · %s · %s\nCreated: %s · Started: %s · Ended: %s\nSession: %s · Turn: %d", run.RunID, run.Status, run.Trigger, taskTime(run.CreatedAt), taskTime(run.StartedAt), taskTime(run.EndedAt), run.SessionID, run.Turn)).FontSize(11).TextColor(muted).Selectable()
					if run.Error != "" {
						ui.Text(c, run.Error).FontSize(11).TextColor(accent).Selectable()
					}
				}
			})
		}
		if s.Review != nil {
			review := s.Review
			cmd := review.Command
			ui.Column(c.Key("routine-review")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, "确认 / Review: "+routineActionLabel(cmd.Action, cmd.EnabledBefore)).FontSize(13).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s\nRoutine: %s (%s) · revision %d", r.HostLabel, s.BotName, cmd.BotID, review.SessionID, review.Name, cmd.ID, cmd.Revision)).FontSize(12).TextColor(fg).Selectable()
				ui.Text(c, "Prompt: "+review.Prompt+"\nSchedule: "+review.Schedule+" · "+review.Timezone).FontSize(12).TextColor(fg).Selectable()
				if cmd.Action == "save" {
					ui.Text(c, fmt.Sprintf("Enabled: %t · Max runs: %s", cmd.Edit.Enabled, routineLimit(cmd.Edit.MaxRuns))).FontSize(12).TextColor(fg)
				}
				ui.Text(c, routineImpact(cmd)).FontSize(12).TextColor(accent)
				if cmd.Action != "delete" {
					routineAdvancedReview(c, r, *review, fg, muted, accent)
				}
				ui.Row(c).Gap(8).Children(func() {
					if r.OnDismiss != nil && ui.Button(c, "返回 / Back to routine").Disabled(s.Busy).Clicked() {
						r.OnDismiss()
					}
					if r.OnConfirm != nil && ui.Button(c, "确认例行任务操作 / Confirm routine action").Disabled(blocked || !s.Loaded || cmd.Revision != s.Revision || dshbotstore.RoutineReviewNeedsWatchAcknowledgement(*review) && !review.WatchAcknowledged).Clicked() {
						r.OnConfirm()
					}
				})
			})
		}
	})
}
func routineLimit(n int) string {
	if n == 0 {
		return "不限 / Unlimited"
	}
	return fmt.Sprint(n)
}
func routineEditor(c *ui.Context, m RoutineModel, blocked bool, fg, muted, accent ui.Color) {
	d := *m.State.Draft
	disabled := blocked || m.State.Review != nil
	ui.Column(c.Key("routine-editor-" + d.ID)).Gap(7).Children(func() {
		ui.Text(c, fmt.Sprintf("本地草稿 / Local draft · revision %d", d.Revision)).FontSize(12).Bold().TextColor(fg)
		if d.Revision != m.State.Revision {
			ui.Text(c, "草稿基线已过期；核对当前记录后可丢弃并重新编辑。 / Draft revision is stale.").FontSize(12).TextColor(accent)
		}
		for _, f := range []struct{ id, label, value string }{{"name", "名称 / Routine name", d.Edit.Name}, {"schedule", "时间规则 / Schedule", d.Edit.Schedule}, {"timezone", "时区 / Timezone", d.Edit.Timezone}, {"maxRuns", "执行次数上限 / Max runs (0 = unlimited)", d.MaxRunsText}} {
			f := f
			value := f.value
			ui.TextInput(c.Key("routine-field-"+f.id), &value).Label(f.label).FillWidth().Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, value)
				}
			})
		}
		ui.Text(c, "规则示例：30m（一次）、every 1h、0 9 * * 1-5；由 Host 校验。").FontSize(11).TextColor(muted)
		prompt := d.Edit.Prompt
		ui.TextArea(c.Key("routine-prompt"), &prompt).Label("例行任务提示词 / Routine prompt").Lines(4, 10).FillWidth().Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("prompt", prompt)
			}
		})
		routineAdvancedEditor(c, m, disabled, fg, muted, accent)
		enabledLabel := "草稿：暂停 / Draft paused"
		if d.Edit.Enabled {
			enabledLabel = "草稿：启用 / Draft enabled"
		}
		if m.OnChange != nil && ui.Button(c, enabledLabel).Disabled(disabled).Clicked() {
			m.OnChange("enabled", fmt.Sprint(!d.Edit.Enabled))
		}
		ui.Row(c).Gap(8).Children(func() {
			if m.OnDiscard != nil && ui.Button(c, "丢弃草稿 / Discard routine draft").Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, routineActionLabel("save", d.Edit.Enabled)).Disabled(disabled || !m.State.Loaded || d.Revision != m.State.Revision).Clicked() {
				m.OnReview("save")
			}
		})
	})
}
