package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

func completedTurnStatus(msg session.Message) string {
	switch readTurnReason(msg).Kind {
	case "error", "failed":
		return "运行失败 / Failed"
	case "cancelled", "canceled", "interrupted", "abort", "aborted":
		return "已停止 / Stopped"
	case "max-tokens":
		return "输出已截断 / Output limit reached"
	default:
		return "已完成 / Completed"
	}
}

func TurnUsageView(c *ui.Context, m Model, msg session.Message, muted ui.Color) {
	if m.Preferences.PerformanceUsage == "compact" {
		return
	}
	u, ok := m.Snapshot.TurnUsage[msg.Seq]
	if !ok {
		return
	}
	key := fmt.Sprintf("turn-usage:%s:%s:%.0f", m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, msg.Seq)
	open := workflowExpanded(m.ThinkingExpanded, key, false)
	arrow := "▸ "
	if open {
		arrow = "▾ "
	}
	label := fmt.Sprintf("本轮用量 / Turn usage · %.0f tokens", u.Total)
	if u.CacheRead != nil && u.Total > u.Output {
		label += fmt.Sprintf(" · 缓存命中 / Cache %.1f%%", 100**u.CacheRead/(u.Total-u.Output))
	}
	if ui.Button(c, arrow+label).FontSize(theme.InterfaceSize(11)).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
		workflowToggle(m.ThinkingExpanded, key, false)
	}
	if !open {
		return
	}
	if len(u.Routes) > 0 {
		ui.Text(c, strings.Join(u.Routes, ", ")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	ui.Text(c, fmt.Sprintf("未缓存输入 / Uncached input %.0f · 输出 / Output %.0f · 总计 / Total %.0f", u.Input, u.Output, u.Total)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	if u.CacheRead != nil {
		ui.Text(c, fmt.Sprintf("缓存读取 / Cache read %.0f", *u.CacheRead)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if u.CacheWrite != nil {
		ui.Text(c, fmt.Sprintf("缓存写入 / Cache write %.0f", *u.CacheWrite)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if u.Reasoning != nil {
		ui.Text(c, fmt.Sprintf("其中思考 / Reasoning subset %.0f", *u.Reasoning)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
}

func PerformanceUsageView(c *ui.Context, snapshot session.ViewState, mode string, muted ui.Color) {
	tokens, ms, steps := session.RecordedPerformance(snapshot.Messages)
	var labels []string
	if ms > 0 {
		labels = append(labels, fmt.Sprintf("输出 / Output %.1f tok/s", tokens*1000/ms))
	}
	var prompt, hit float64
	for _, source := range snapshot.CostSources {
		if source.ID != snapshot.SelectedID || source.Usage == nil {
			continue
		}
		for _, route := range source.Usage.Models {
			if route.Peak != nil {
				prompt += route.Peak.Miss + route.Peak.Hit
				hit += route.Peak.Hit
			}
			if route.OffPeak != nil {
				prompt += route.OffPeak.Miss + route.OffPeak.Hit
				hit += route.OffPeak.Hit
			}
		}
	}
	if prompt > 0 {
		labels = append(labels, fmt.Sprintf("缓存命中 / Cache %.1f%%", 100*hit/prompt))
	}
	if len(labels) == 0 {
		return
	}
	row := ui.Column(c.Key("performance:"+snapshot.PlanHistoryBinding+":"+snapshot.SelectedID)).FillWidth().Padding(4, 8).Gap(4)
	open := ui.Local(row, "open", func() bool { return false })
	row.Children(func() {
		text := strings.Join(labels, " · ")
		if mode == "compact" {
			ui.Text(c, text).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			return
		}
		if ui.Button(c, text+" ▾").FontSize(theme.InterfaceSize(11)).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
			*open = !*open
		}
		if *open {
			ui.Text(c, fmt.Sprintf("已加载记录 / Loaded history · %d 轮 / turns · %d 步 / steps", len(snapshot.TurnUsage), steps)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			if ms > 0 {
				ui.Text(c, fmt.Sprintf("有解码时间的输出 / Timed output %.0f tokens · %.0f ms", tokens, ms)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			if prompt > 0 {
				ui.Text(c, fmt.Sprintf("Host 会话输入 / Session input %.0f · 缓存读取 / Cache read %.0f", prompt, hit)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
		}
	})
}
