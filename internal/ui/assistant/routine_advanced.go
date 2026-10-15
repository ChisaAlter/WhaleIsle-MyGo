package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"slices"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

const routineWatchAcknowledgement = "我理解命令影响及目录回退，允许按此配置自动执行 / I understand command effects and working-directory fallback"

func routineContextName(id string, rows []pluginrpc.DshbotRoutine) string {
	if id == "self" {
		return "本任务上一轮输出 / This routine's previous output (self)"
	}
	for _, r := range rows {
		if r.ID == id {
			return r.Name + " (" + id + ")"
		}
	}
	return "来源已不可用；保存前须明确移除 / Unavailable source; remove explicitly before saving: " + id
}
func routineAdvancedEditor(c *ui.Context, m RoutineModel, disabled bool, fg, muted, accent ui.Color) {
	d := m.State.Draft.Advanced
	ui.Column(c.Key("routine-advanced")).Gap(6).Children(func() {
		ui.Text(c, "高级配置 / Advanced settings").FontSize(12).Bold().TextColor(fg)
		ui.Text(c, "仅提交明确修改的高级字段；未改字段由 Host 保留。不会探测 URL、试运行命令或读取触发凭据。").FontSize(11).TextColor(muted)
		choices := []string{"self"}
		for _, r := range m.State.Routines {
			if r.BotID == m.State.Scope.BotID && r.ID != m.State.SelectedID {
				choices = append(choices, r.ID)
			}
		}
		for _, id := range d.ContextFrom {
			if !slices.Contains(choices, id) {
				choices = append(choices, id)
			}
		}
		ui.Text(c, "上下文来源 / Context sources (最多 3 项 / max 3)").FontSize(12).TextColor(fg)
		for _, id := range choices {
			id := id
			checked := slices.Contains(d.ContextFrom, id)
			if ui.Checkbox(c.Key("routine-context-"+id), &checked, routineContextName(id, m.State.Routines)).Disabled(disabled || !checked && len(d.ContextFrom) >= 3).Changed() && m.OnChange != nil {
				m.OnChange("contextToggle", id)
			}
		}
		silent := d.SilentAllowed
		if ui.Checkbox(c, &silent, "允许静默完成 / Allow silent response").Disabled(disabled).Changed() && m.OnChange != nil {
			m.OnChange("silentAllowed", fmt.Sprint(silent))
		}
		ui.Text(c, "静默只影响正常输出方式，不表示任务不会执行或没有外部影响。").FontSize(11).TextColor(muted)
		if d.WatchUnknown {
			ui.Text(c, "未知 watch 配置保持原样；明确选择类型才替换 / Unknown watch preserved: "+d.WatchRaw).FontSize(11).TextColor(accent).Selectable()
		}
		ui.Row(c).Gap(8).Children(func() {
			for _, choice := range []struct{ id, label string }{{"", "不检查 / No watch"}, {"url", "读取 URL / URL watch"}, {"command", "执行命令 / Command watch"}} {
				choice := choice
				if ui.Button(c, choice.label).Disabled(disabled || !d.WatchUnknown && d.WatchKind == choice.id).Clicked() && m.OnChange != nil {
					m.OnChange("watchKind", choice.id)
				}
			}
		})
		if !d.WatchUnknown && d.WatchKind != "" {
			value, timeout := d.WatchValue, d.WatchTimeout
			label := "完整 URL / Full watch URL"
			if d.WatchKind == "command" {
				label = "完整 Host shell 命令 / Full Host shell command"
			}
			ui.TextArea(c.Key("routine-watch-value"), &value).Label(label).Lines(2, 6).FillWidth().Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange("watchValue", value)
				}
			})
			ui.TextInput(c.Key("routine-watch-timeout"), &timeout).Label("超时毫秒 / Timeout ms (1–300000)").FillWidth().Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange("watchTimeout", timeout)
				}
			})
			ui.Text(c, "检查结果未变化时可跳过调度；检查失败仍可能排队执行 Bot。保存本身不执行检查。").FontSize(11).TextColor(accent)
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
		sources = []string{"无 / None"}
	}
	ui.Text(c, "有效上下文来源 / Effective context sources:\n"+strings.Join(sources, "\n")+fmt.Sprintf("\nSilent allowed: %t", d.SilentAllowed)).FontSize(12).TextColor(fg).Selectable()
	ui.Text(c, "后续执行会将所选任务的上一轮输出提供给此 Bot（每份最多 8000 字符）；self 是本任务上一轮输出。静默不取消执行、工具或消息影响。").FontSize(11).TextColor(muted)
	if d.WatchUnknown {
		ui.Text(c, "有效 watch：未知配置，未编辑保持原样 / Effective watch is unrecognized and preserved:\n"+d.WatchRaw).FontSize(12).TextColor(accent).Selectable()
	} else if d.WatchKind == "" {
		ui.Text(c, "有效 watch / Effective watch: none").FontSize(12).TextColor(fg)
	} else {
		ui.Text(c, fmt.Sprintf("有效 watch / Effective watch: %s\n完整目标 / Full target:\n%s\nTimeout: %s ms", d.WatchKind, d.WatchValue, d.WatchTimeout)).FontSize(12).TextColor(fg).Selectable()
		if d.WatchKind == "url" {
			ui.Text(c, "自动调度前 Host 会读取该 URL（会跟随重定向），内容可能作为此 Bot 的输入。").FontSize(11).TextColor(accent)
		}
	}
	if d.WatchKind == "command" || d.WatchUnknown {
		workspace := r.WorkspaceID
		if workspace == "" {
			workspace = "（未配置 / not configured）"
		}
		ui.Text(c, "配置 workspaceId（未验证实际 cwd）: "+workspace+"\n运行时目录规则：若该路径存在且是目录则使用；否则用 DSH_HOME，未设时用 DSHD_HOME；均未设则继承 Host 进程目录。实际执行目录此刻无法核实，且运行时可能变化。\ncommand 使用 Host 的 cmd（Windows）或 sh（其他系统）执行，可能读写文件、启动程序、访问网络或发送信息。").FontSize(12).TextColor(accent).Selectable()
	}
	if d.WatchKind != "" || d.WatchUnknown {
		ui.Text(c, "自动调度的 watch 检查失败仍可能排队执行 Bot；未变化才可跳过。手动立即运行会跳过 watch，但仍使用所选上下文。").FontSize(11).TextColor(accent)
	}
	if dshbotstore.RoutineReviewNeedsWatchAcknowledgement(r) {
		ack := r.WatchAcknowledged
		if ui.Checkbox(c, &ack, routineWatchAcknowledgement).Disabled(m.State.Busy).Changed() && m.OnAcknowledgeWatch != nil {
			m.OnAcknowledgeWatch(ack)
		}
	}
}
