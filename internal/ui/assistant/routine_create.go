package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
)

type RoutineCreateModel struct {
	State                                                                   dshbotstore.RoutineCreateSnapshot
	HostLabel, BotName                                                      string
	OnStart, OnReview, OnConfirm, OnDismiss, OnDiscard, OnRebase, OnRefresh func()
	OnChange                                                                func(field, value string)
	OnAdopt                                                                 func(id string)
}

func routineCreateImpact(enabled bool) string {
	if enabled {
		return "将创建并启用每日例行任务。Host 会计算下次时间并自动调度，Bot 可能执行工具、发送外部消息并产生模型费用；本次保存不会额外调用立即运行。"
	}
	return "将创建暂停的每日例行任务，后续须单独确认启用才会自动调度。本次保存不会调用立即运行；启用后可能执行工具、发送外部消息并产生模型费用。"
}
func routineCreateBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	w := m.RoutineCreate
	s := w.State
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || w.OnStart == nil {
		return
	}
	blocked := m.Busy || m.PluginRPCBusy || s.Busy || !m.HarnessReady || !m.DshbotEnabled
	ui.Column(c.Key("routine-create")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "新建 Bot 例行任务 / Create Bot routine").FontSize(13).Bold().TextColor(fg)
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(12).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted).Selectable()
		}
		if s.Draft == nil {
			if ui.Button(c, "新建例行任务 / New routine").Disabled(blocked || !s.Loaded || s.Scope.BotID == "").Clicked() {
				w.OnStart()
			}
			if !s.Loaded {
				ui.Text(c, "先刷新上方例行任务列表；空列表也可创建 / Refresh the routine catalog first; empty catalogs are supported").FontSize(11).TextColor(muted)
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		ui.Text(c, "每日任务 / Daily · 默认09:00、暂停、次数不限；不设置watch或跨任务上下文。").FontSize(11).TextColor(muted)
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · revision %d", w.HostLabel, w.BotName, s.Scope.BotID, d.Revision)).FontSize(11).TextColor(muted).Selectable()
		for _, f := range []struct{ id, label, value string }{{"name", "新任务名称 / New routine name", d.Name}, {"hour", "小时 / Hour (00–23)", d.Hour}, {"minute", "分钟 / Minute (00–59)", d.Minute}, {"timezone", "时区 / IANA timezone (例如 Asia/Shanghai)", d.Timezone}, {"maxRuns", "执行次数上限 / Max runs (0 = unlimited)", d.MaxRuns}} {
			f := f
			value := f.value
			ui.TextInput(c.Key("create-"+f.id), &value).Label(f.label).FillWidth().Disabled(disabled).OnChange(func() {
				if w.OnChange != nil {
					w.OnChange(f.id, value)
				}
			})
		}
		ui.Text(c, "原版按浏览器本地时区安排；无法识别本机IANA时区时须明确填写，不会猜测。 / Specify the intended IANA timezone.").FontSize(11).TextColor(muted)
		prompt := d.Prompt
		ui.TextArea(c.Key("create-prompt"), &prompt).Label("新任务提示词 / New routine prompt").Lines(4, 10).FillWidth().Disabled(disabled).OnChange(func() {
			if w.OnChange != nil {
				w.OnChange("prompt", prompt)
			}
		})
		label := "新任务：暂停 / New routine paused"
		if d.Enabled {
			label = "新任务：启用 / New routine enabled"
		}
		if w.OnChange != nil && ui.Button(c, label).Disabled(disabled).Clicked() {
			w.OnChange("enabled", fmt.Sprint(!d.Enabled))
		}
		if s.Uncertain {
			ui.Text(c, "同一创建请求已锁定。刷新只查询目录，不会重复提交；核对新记录后可采用它。 / This creation is locked until its outcome is checked.").FontSize(12).TextColor(accent)
			if w.OnRefresh != nil && ui.Button(c, "核对创建结果 / Check creation outcome").Disabled(blocked).Clicked() {
				w.OnRefresh()
			}
			for _, row := range s.Candidates {
				row := row
				ui.Column(c.Key("create-candidate-" + row.ID)).Background(field).Padding(10).Gap(6).Children(func() {
					ui.Text(c, fmt.Sprintf("候选记录 / New record: %s (%s)\n%s\n%s · %s · enabled %t", row.Name, row.ID, row.Prompt, row.Schedule, row.Timezone, row.Enabled)).FontSize(12).TextColor(fg).Selectable()
					if w.OnAdopt != nil && ui.Button(c, "采用这条记录 / Use record: "+row.ID).Disabled(blocked).Clicked() {
						w.OnAdopt(row.ID)
					}
				})
			}
			if s.RetryVerified && w.OnRebase != nil && ui.Button(c, "未发现新记录，重新审阅 / No new record; review again").Disabled(blocked).Clicked() {
				w.OnRebase()
			}
		} else {
			ui.Row(c).Gap(8).Children(func() {
				if w.OnDiscard != nil && ui.Button(c, "放弃新建草稿 / Discard new draft").Disabled(blocked).Clicked() {
					w.OnDiscard()
				}
				if w.OnRebase != nil && ui.Button(c, "采用当前目录基线 / Recheck catalog").Disabled(blocked || !s.Loaded).Clicked() {
					w.OnRebase()
				}
				if w.OnReview != nil && ui.Button(c, "审阅新建… / Review creation…").Disabled(disabled || !s.Loaded).Clicked() {
					w.OnReview()
				}
			})
		}
		if s.Review != nil {
			cmd := s.Review.Command
			ui.Column(c.Key("create-review")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, "确认创建 / Review creation").FontSize(13).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s\nName: %s · revision %d\nPrompt: %s\nSchedule: %s · %s\nEnabled: %t · Max runs: %s", w.HostLabel, w.BotName, cmd.BotID, cmd.SessionID, cmd.Edit.Name, cmd.Revision, cmd.Edit.Prompt, cmd.Edit.Schedule, cmd.Edit.Timezone, cmd.Edit.Enabled, routineLimit(cmd.Edit.MaxRuns))).FontSize(12).TextColor(fg).Selectable()
				ui.Text(c, routineCreateImpact(cmd.Edit.Enabled)).FontSize(12).TextColor(accent)
				ui.Text(c, "无watch、无跨任务上下文，允许静默；Host保存后生成任务ID。 / No watch or cross-routine context; silent responses allowed.").FontSize(11).TextColor(muted)
				ui.Row(c).Gap(8).Children(func() {
					if w.OnDismiss != nil && ui.Button(c, "返回修改 / Back to draft").Disabled(s.Busy).Clicked() {
						w.OnDismiss()
					}
					if w.OnConfirm != nil && ui.Button(c, "确认保存新例行任务 / Confirm new routine").Disabled(blocked || !s.Loaded).Clicked() {
						w.OnConfirm()
					}
				})
			})
		}
	})
}
