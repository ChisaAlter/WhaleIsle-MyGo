package assistant

import (
	"fmt"
	"strings"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
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
		return "暂停任务 / Pause"
	case "resume":
		return "恢复任务 / Resume"
	case "cancel":
		return "取消任务 / Cancel task"
	case "retry":
		return "重试任务 / Retry task"
	case "stop":
		return "停止 Bot 会话 / Stop Bot"
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
			ui.Text(c, "Bot 任务 / Bot tasks").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, "刷新任务 / Refresh tasks").Disabled(blocked || s.Scope.BotID == "").Clicked() {
				tasks.OnRefresh()
			}
		})
		ui.Text(c, "仅显示所选 Bot 发出或收到的委派任务；刷新读取当前 Host，操作由你确认后执行。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if s.Scope.BotID == "" {
			ui.Text(c, "先选择当前 Host 的 Bot / Select a Bot first").TextColor(muted)
			return
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if s.Busy {
			ui.Text(c, "正在等待 Host… / Waiting for Host…").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if !s.Loaded {
			ui.Text(c, "点击刷新加载任务。上次读取失败时不会开放写操作。 / Refresh before making changes.").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Loaded {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("%s · %s · revision %d", s.BotName, s.Scope.BotID, s.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Grow(1)
				if tasks.OnReview != nil && ui.Button(c, taskActionLabel("stop")).Disabled(blocked).Clicked() {
					tasks.OnReview("stop", "")
				}
			})
		}
		if s.Loaded && len(s.Tasks) == 0 {
			ui.Text(c, "此 Bot 暂无委派任务 / No delegation tasks for this Bot").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		filterKey := s.Scope.HostKey + ":" + s.Scope.BotID
		query := ui.Local(block, "task-query:"+filterKey, func() string { return "" })
		status := ui.Local(block, "task-status:"+filterKey, func() string { return "all" })
		ui.TextInput(c.Key("task-search"), query).Label("搜索任务、参与者或错误 / Search tasks, participants, or errors").FillWidth()
		ui.Select(c.Key("task-status"), status, []string{"all", "queued", "delivered", "completed", "failed", "paused", "cancelled"}).Label("任务状态 / Task status").FillWidth()
		visible := visibleBotTasks(s.Tasks, *query, *status)
		if s.Loaded && len(s.Tasks) > 0 && len(visible) == 0 {
			ui.Text(c, "没有匹配的任务 / No matching tasks").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		for _, task := range visible {
			task := task
			ui.Column(c.Key("bot-task-"+task.ID)).Background(field).Border(1, border).Radius(8).Padding(10).Gap(5).Children(func() {
				title := task.ID + " · " + task.Status
				if s.SelectedID == task.ID {
					title = "▾ " + title
				}
				if tasks.OnSelect != nil && ui.Button(c, title).Disabled(blocked).Clicked() {
					tasks.OnSelect(task.ID)
				}
				ui.Text(c, taskParticipant(task.FromName, task.FromID)+" → "+taskParticipant(task.ToName, task.ToID)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Text(c, clip(task.Task, 180)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				if s.SelectedID != task.ID {
					return
				}
				ui.Row(c).Gap(6).Children(func() {
					for _, participant := range []struct{ id, label string }{
						{task.FromID, "打开发件人会话 / Open sender conversation"},
						{task.ToID, "打开收件人会话 / Open recipient conversation"},
					} {
						participant := participant
						if m.Contacts.State.Loaded && m.OnDshbotOpenRosterSession != nil {
							for _, item := range m.Contacts.State.Catalog.Items {
								if item.ID == strings.TrimSpace(participant.id) && item.Kind == "bot" && strings.TrimSpace(item.SessionID) != "" {
									if ui.Button(c, participant.label).Disabled(blocked || m.Contacts.State.Busy).Clicked() {
										m.OnDshbotOpenRosterSession(item.ID)
									}
									break
								}
							}
						}
					}
				})
				ui.Text(c, "任务 / Task: "+task.Task).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				if len(task.Constraints) > 0 {
					ui.Text(c, "约束 / Constraints:\n- "+strings.Join(task.Constraints, "\n- ")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				if task.SuccessCriteria != "" {
					ui.Text(c, "成功标准 / Success criteria: "+task.SuccessCriteria).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				if task.ResultSummary != "" {
					ui.Text(c, "结果 / Result: "+task.ResultSummary).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				if task.Error != "" {
					ui.Text(c, "错误 / Error: "+task.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				}
				ui.Text(c, fmt.Sprintf("Attempts: %d · %s", task.Attempts, taskTime(task.UpdatedAt))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if task.ParentTaskID != "" || task.RetryOf != "" {
					ui.Text(c, "重试源 / Retry of: "+firstTaskValue(task.ParentTaskID, task.RetryOf)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
				}
				ui.Row(c).Gap(6).Children(func() {
					for _, action := range []string{"pause", "resume", "cancel", "retry"} {
						action := action
						if pluginrpc.DshbotTaskActionAllowed(action, task.Status) && tasks.OnReview != nil && ui.Button(c, taskActionLabel(action)).Disabled(blocked || !s.Loaded).Clicked() {
							tasks.OnReview(action, task.ID)
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
			ui.Column(c.Key("bot-task-review")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, "确认："+taskActionLabel(command.Action)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Bot: %s (%s)\nTask: %s · revision %d", s.BotName, command.BotID, firstTaskValue(command.TaskID, "—"), command.Revision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				caveat := "仅修改任务状态；不会撤销已发生的工具、文件或外部操作。"
				if command.Action == "retry" {
					caveat = "会创建新的任务尝试并唤醒接收 Bot，可能再次执行工具或产生模型费用。原任务保留。"
				}
				if command.Action == "resume" {
					caveat = "会将任务重新排队并唤醒接收 Bot，可能执行工具或产生模型费用。"
				}
				if command.Action == "stop" {
					caveat = "会停止此 Bot 的当前会话（群组会停止成员轮次）；保留 inbox，不取消全部委派任务，也不撤销已发生的操作。"
				}
				ui.Text(c, caveat).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				if command.Action == "retry" {
					ui.Text(c, "失败后再次确认会沿用同一请求 ID；不要重复新建任务。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				ui.Row(c).Gap(8).Children(func() {
					if tasks.OnDismiss != nil && ui.Button(c, "返回 / Back").Disabled(s.Busy).Clicked() {
						tasks.OnDismiss()
					}
					if tasks.OnConfirm != nil && ui.Button(c, "确认执行 / Confirm action").Disabled(blocked || !s.Loaded || command.Revision != s.Revision).Clicked() {
						tasks.OnConfirm()
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
