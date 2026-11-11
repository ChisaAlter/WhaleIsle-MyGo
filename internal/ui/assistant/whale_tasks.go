package assistant

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type WhaleTasksModel struct {
	State                                                  assiststore.WhaleTaskSnapshot
	OnRefresh, OnStart, OnPickFiles, OnAuthorize, OnRevoke func()
	OnEdit                                                 func(string, string)
	OnControl                                              func(string, string)
	OnOpenSession                                          func(string)
}

func whaleTaskCopy(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}
func whaleTaskState(state string) string {
	zh := map[string]string{"queued": "等待执行", "running": "整理中", "needs_user": "需要补充信息", "stopping": "正在停止", "cancelled": "已停止", "completed": "已完成", "failed": "失败", "interrupted": "重启后待继续", "unknown": "结果待核对", "pending": "待投递", "sending": "投递中", "sent": "已投递"}
	if text := zh[state]; text != "" {
		return whaleTaskCopy(text, state)
	}
	return state
}
func whaleTasksBlock(c *ui.Context, m Model, fg, muted, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeWhale || m.WhaleTasks.OnRefresh == nil {
		return
	}
	w := m.WhaleTasks
	s := w.State
	blocked := s.Busy || !m.HarnessReady || !m.WhaleEnabled
	root := ui.Column(c.Key("whale-background-tasks")).FillWidth().MinWidth(0).Gap(10)
	lastNotice := ui.Local(root, "last-notice", func() string { return "" })
	if s.Notice != "" && *lastNotice != s.Notice {
		c.AddToast(ui.Toast{ID: "whale-background-task-notice", Description: s.Notice, Timeout: 4 * time.Second})
	}
	*lastNotice = s.Notice
	root.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, whaleTaskCopy("后台资料整理", "Background reading")).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(fg).Grow(1)
			if ui.Button(c, whaleTaskCopy("刷新任务", "Refresh tasks")).Disabled(blocked).Clicked() {
				w.OnRefresh()
			}
		})
		ui.Text(c, whaleTaskCopy("选定文本资料与目标。结果会回到鲸鱼娘会话，关闭此面板仍会继续。", "Choose text files and a goal. Results return to the Whale conversation while this panel is closed.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.CreateUnknown && !s.Busy {
			ui.Text(c, whaleTaskCopy("提交结果尚未确认，请刷新任务核对后再提交。", "Submission is not confirmed. Refresh and check tasks before submitting again.")).TextColor(accent)
		}
		if s.Loaded && !s.Catalog.Available {
			message := s.Catalog.Error
			if message == "" {
				message = whaleTaskCopy("当前 Host 未提供后台资料任务。", "This Host does not provide background reading tasks.")
			}
			ui.Text(c, message).TextColor(muted)
			return
		}
		field := func(label, key, value string, lines int) {
			ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			ui.TextArea(c.Key("whale-task-"+key), &value).Label(label).FillWidth().MinWidth(0).Lines(lines, lines+2).Disabled(blocked).OnChange(func() {
				if w.OnEdit != nil {
					w.OnEdit(key, value)
				}
			})
		}
		active := false
		for _, task := range s.Catalog.Tasks {
			active = active || task.Active() || task.Delivering()
		}
		form := ui.Column(c.Key("whale-task-create")).FillWidth().MinWidth(0).Gap(8)
		creating := ui.Local(form, "open", func() bool { return len(s.Catalog.Tasks) == 0 || s.Goal != "" })
		seen := ui.Local(form, "count", func() int { return len(s.Catalog.Tasks) })
		if len(s.Catalog.Tasks) > *seen && !s.Busy && s.Goal == "" {
			*creating = false
		}
		*seen = len(s.Catalog.Tasks)
		form.Children(func() {
			if ui.Button(c, whaleTaskCopy("新建资料任务", "New reading task")).Clicked() {
				*creating = !*creating
			}
			if !*creating {
				return
			}
			field(whaleTaskCopy("整理目标", "Reading goal"), "goal", s.Goal, 2)
			field(whaleTaskCopy("资料路径（每行一个）", "Input paths (one per line)"), "files", s.Files, 2)
			if w.OnPickFiles != nil && ui.Button(c, whaleTaskCopy("选择资料", "Choose files")).Disabled(blocked).Clicked() {
				w.OnPickFiles()
			}
			ui.Text(c, whaleTaskCopy("支持 UTF-8 文本：TXT、Markdown、CSV、JSON、LOG、RST。最多 8 个文件，单个 100KB、合计 300KB。", "UTF-8 text: TXT, Markdown, CSV, JSON, LOG, RST. Up to 8 files, 100KB each, 300KB total.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			field(whaleTaskCopy("完成条件（可选，每行一项）", "Completion criteria (optional, one per line)"), "criteria", s.Criteria, 2)
			if ui.Button(c, whaleTaskCopy("开始整理", "Start reading")).Disabled(blocked || !s.Loaded || !s.Catalog.Available || active || s.CreateUnknown || strings.TrimSpace(s.Goal) == "" || strings.TrimSpace(s.Files) == "").Clicked() && w.OnStart != nil {
				w.OnStart()
			}
		})
		for _, task := range s.Catalog.Tasks {
			whaleTaskRow(c, w, task, blocked, fg, muted, border, accent)
		}
		owner := ui.Column(c.Key("whale-task-im-owner")).FillWidth().Gap(8)
		open := ui.Local(owner, "open", func() bool { return false })
		owner.Children(func() {
			if ui.Button(c, whaleTaskCopy("飞书后台任务授权", "Feishu task authorization")).Clicked() {
				*open = !*open
			}
			if !*open {
				return
			}
			ui.Text(c, whaleTaskCopy("指定允许通过 /task 提交资料的人；资料仅限该条飞书消息的附件。", "Choose who may submit /task requests. Each request is limited to its attached files.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			field(whaleTaskCopy("飞书机器人 ID", "Feishu bot ID"), "bot", s.BotID, 1)
			field(whaleTaskCopy("授权人的 open_id", "Authorized person's open_id"), "actor", s.Actor, 1)
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, whaleTaskCopy("授权此人", "Authorize person")).Disabled(blocked || !s.Catalog.Available || strings.TrimSpace(s.BotID) == "" || strings.TrimSpace(s.Actor) == "" || s.Actor == "*").Clicked() && w.OnAuthorize != nil {
					w.OnAuthorize()
				}
				if ui.Button(c, whaleTaskCopy("撤销此机器人授权", "Revoke bot authorization")).Disabled(blocked || !s.Catalog.Available || strings.TrimSpace(s.BotID) == "").Clicked() && w.OnRevoke != nil {
					w.OnRevoke()
				}
			})
		})
	})
}
func whaleTaskRow(c *ui.Context, m WhaleTasksModel, task pluginrpc.WhaleTask, blocked bool, fg, muted, border, accent ui.Color) {
	ui.Column(c.Key("whale-task-result-"+task.ID)).FillWidth().MinWidth(0).Gap(6).Padding(10, 0).Children(func() {
		ui.Box(c).FillWidth().Height(1).Background(border)
		ui.Text(c, task.Goal).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg).Selectable()
		ui.Text(c, whaleTaskState(task.State)+" · "+task.Model.Provider+" / "+task.Model.Model).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		var names []string
		for _, input := range task.Inputs {
			names = append(names, input.Name)
		}
		ui.Text(c, whaleTaskCopy("资料：", "Inputs: ")+strings.Join(names, " · ")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if task.Error != "" {
			ui.Text(c, task.Error).TextColor(accent).Selectable()
		}
		if task.Result != nil {
			ui.Text(c, task.Result.Summary).FontSize(theme.InterfaceSize(13)).TextColor(fg).Selectable()
			for _, reference := range task.Result.Evidence {
				for _, input := range task.Inputs {
					if input.ID == reference {
						ui.Text(c, whaleTaskCopy("来源：", "Source: ")+input.Name).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
				}
			}
			if len(task.Result.Unmet) > 0 {
				ui.Text(c, whaleTaskCopy("尚未满足：", "Unmet: ")+strings.Join(task.Result.Unmet, "；")).TextColor(accent).Selectable()
			}
		}
		for _, effect := range task.Effects {
			ui.Text(c, fmt.Sprintf("%s · %s", whaleTaskCopy(map[string]string{"session": "会话", "pet": "桌宠", "im": "飞书"}[effect.Target], effect.Target), whaleTaskState(effect.State))).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if effect.Error != "" {
				ui.Text(c, effect.Error).TextColor(accent)
			}
		}
		ui.Row(c).Gap(8).Children(func() {
			if task.Active() && task.State != "stopping" && ui.Button(c, whaleTaskCopy("停止任务", "Stop task")).Disabled(blocked).Clicked() && m.OnControl != nil {
				m.OnControl(task.ID, "stop")
			}
			if task.CanResume() && ui.Button(c, whaleTaskCopy("继续任务", "Resume task")).Disabled(blocked).Clicked() && m.OnControl != nil {
				m.OnControl(task.ID, "resume")
			}
			if task.Execution != nil && task.Execution.SessionID != "" && m.OnOpenSession != nil && ui.Button(c, whaleTaskCopy("查看执行记录", "Open execution history")).Clicked() {
				m.OnOpenSession(task.Execution.SessionID)
			}
		})
	})
}
