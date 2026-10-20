package sidebar

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func sessionScheduleMark(c *ui.Context, count int, ink ui.Color) {
	mark := ui.Box(c).Label(fmt.Sprintf(workspaceText("%d 个定时任务", "%d scheduled tasks"), count)).Size(16, 20).Shrink(0)
	// Clickable but not focusable: pressing the passive mark consumes the
	// pointer event and never activates the Session through its parent row.
	_ = mark.Clicked()
	mark.Draw(func(p *ui.Painter, r ui.Rect) {
		var clock, hand ui.Path
		clock.Circle(r.X+8, r.Y+10, 6)
		hand.MoveTo(r.X+8, r.Y+6).LineTo(r.X+8, r.Y+10).LineTo(r.X+11, r.Y+12)
		p.StrokePath(&clock, 1.2, ink)
		p.StrokePath(&hand, 1.2, ink)
	})
}

func orderSchedulePreview(rows []schedstore.Entry, now time.Time) []schedstore.Entry {
	ordered := append([]schedstore.Entry(nil), rows...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, le := time.Parse(time.RFC3339Nano, ordered[i].ScheduledAt)
		right, re := time.Parse(time.RFC3339Nano, ordered[j].ScheduledAt)
		leftOverdue, rightOverdue := le == nil && !left.After(now), re == nil && !right.After(now)
		if leftOverdue != rightOverdue {
			return leftOverdue
		}
		return le == nil && re == nil && left.Before(right)
	})
	return ordered
}

func schedulePreviewTiming(row schedstore.Entry, now time.Time) string {
	at, err := time.Parse(time.RFC3339Nano, row.ScheduledAt)
	if err != nil {
		return workspaceText("下次运行时间不可用", "Next run unavailable")
	}
	difference := at.Sub(now).Seconds()
	relative := workspaceText("现在", "due now")
	if difference != 0 {
		for _, unit := range []struct {
			seconds float64
			zh, en  string
		}{{86400, "天", "day"}, {3600, "小时", "hour"}, {60, "分钟", "minute"}, {1, "秒", "second"}} {
			if math.Abs(difference) < unit.seconds && unit.seconds != 1 {
				continue
			}
			value := math.Floor(math.Abs(difference) / unit.seconds)
			if difference > 0 {
				value = math.Ceil(math.Abs(difference) / unit.seconds)
			}
			value = math.Max(1, value)
			name := unit.en
			if value != 1 {
				name += "s"
			}
			if difference > 0 {
				relative = fmt.Sprintf(workspaceText("%.0f%s后", "in %.0f %s"), value, workspaceText(unit.zh, name))
			} else {
				relative = fmt.Sprintf(workspaceText("已逾期 %.0f%s", "%.0f %s overdue"), value, workspaceText(unit.zh, name))
			}
			break
		}
	}
	return at.In(time.Local).Format("2006-01-02 15:04:05") + " (" + relative + ")"
}

func schedulePreviewFrequency(row schedstore.Entry) string {
	var rule struct {
		Kind, Time, TimeZone, Expression string
		EverySeconds                     int
		Weekdays                         []int
	}
	_ = json.Unmarshal(row.Raw, &rule)
	switch rule.Kind {
	case "every":
		for _, unit := range []struct {
			seconds int
			zh, en  string
		}{{86400, "天", "days"}, {3600, "小时", "hours"}, {60, "分钟", "minutes"}, {1, "秒", "seconds"}} {
			if rule.EverySeconds > 0 && rule.EverySeconds%unit.seconds == 0 {
				return fmt.Sprintf(workspaceText("每 %d %s", "Every %d %s"), rule.EverySeconds/unit.seconds, workspaceText(unit.zh, unit.en))
			}
		}
	case "daily":
		return workspaceText("每天 ", "Every day ") + rule.Time + " " + rule.TimeZone
	case "weekly":
		return fmt.Sprintf(workspaceText("每周 %v · %s %s", "Weekly %v · %s %s"), rule.Weekdays, rule.Time, rule.TimeZone)
	case "cron":
		return workspaceText("自定义 ", "Custom ") + rule.Expression + " " + rule.TimeZone
	}
	return workspaceText("仅一次", "Once")
}

func sessionScheduleHover(c *ui.Context, m Model, anchor ui.Element, item session.Item, status string, rows []schedstore.Entry, epoch uint64, colors theme.Colors) {
	state := ui.Local(anchor, struct {
		epoch   uint64
		session string
	}{epoch, item.SessionID}, func() struct {
		shown bool
		now   time.Time
	} {
		return struct {
			shown bool
			now   time.Time
		}{}
	})
	shown := false
	ui.TooltipBase(c.Key("session-task-hover"), anchor, func(tip ui.Element) {
		shown = true
		if !state.shown {
			state.now = time.Now()
		}
		state.shown = true
		ordered := orderSchedulePreview(rows, state.now)
		width, height := m.viewport(c)
		tip.Label("Scheduled task preview: "+item.SessionID).Width(min(float32(300), max(float32(0), width-24))).MaxHeight(max(float32(0), height-24)).Clip().Padding(12).Radius(12).Background(chrome.Color(colors.BG)).Border(.5, chrome.Color(colors.Border)).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, .16)).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, item.Title).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.FG)).FillWidth().MinWidth(0).MaxLines(2)
				for _, task := range ordered[:min(2, len(ordered))] {
					ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
						ui.Text(c, task.Title).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG)).FillWidth().MinWidth(0).MaxLines(2)
						ui.Text(c, schedulePreviewFrequency(task)+" · "+schedulePreviewTiming(task, state.now)).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted)).FillWidth().MinWidth(0).MaxLines(3)
					})
				}
				if omitted := len(ordered) - 2; omitted > 0 {
					ui.Text(c, fmt.Sprintf(workspaceText("另有 %d 个任务", "%d more tasks"), omitted)).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted))
				}
				if text := statusLabel(status); text != "" {
					ui.Text(c, text).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted))
				}
			})
		})
	})
	if !shown {
		state.shown = false
	}
}
