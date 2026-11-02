package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// turnStatusKind is the stable terminal state of one Turn. Comparisons stay on
// this value; localized copy is only ever rendered, never compared.
func turnStatusKind(msg session.Message) string {
	switch readTurnReason(msg).Kind {
	case "error", "failed":
		return "failed"
	case "cancelled", "canceled", "interrupted", "abort", "aborted":
		return "stopped"
	case "max-tokens":
		return "truncated"
	default:
		return "done"
	}
}

// turnStatusLabel is the row title for a finished Turn.
func turnStatusLabel(msg session.Message) string {
	switch turnStatusKind(msg) {
	case "failed":
		return i18n.T(i18n.KeyChatTurnFailed)
	case "stopped":
		return i18n.T(i18n.KeyChatTurnStopped)
	case "truncated":
		return i18n.T(i18n.KeyChatTurnTruncated)
	default:
		// The store writes the elapsed milliseconds into the event body; the
		// Original strip reads 「已完成，用时 …」 from the same figure.
		return formatTurnStatus(msg.Body)
	}
}

func turnStatusRowStatus(msg session.Message) rowStatus {
	switch turnStatusKind(msg) {
	case "failed":
		return statusError
	case "stopped", "truncated":
		return statusWarning
	default:
		return statusDone
	}
}

// completedTurnStatus keeps the legacy helper for callers that only need the
// finished/unfinished distinction. It returns a stable kind, not localized copy.
func completedTurnStatus(msg session.Message) string {
	return turnStatusKind(msg)
}

// TurnUsageView renders the per-Turn token usage as the Original
// TurnUsageDisclosure: one disclosure row inside the message action area whose
// summary is the compact total and whose body lists the exact buckets.
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
	summary := fmt.Sprintf("%s %s", formatTokensCompact(u.Total), i18n.T(i18n.KeyChatUsageTokens))
	if u.CacheRead != nil && u.Total > u.Output {
		summary += " · " + fmt.Sprintf(i18n.T(i18n.KeyChatUsageCachePct), 100**u.CacheRead/(u.Total-u.Output))
	}
	root := ui.Column(c).FillWidth().MinWidth(0)
	root.Children(func() {
		paintDisclosureRow(c, disclosureRow{
			Key:        key,
			Icon:       disclosureIconData,
			Title:      i18n.T(i18n.KeyChatUsageTitle),
			Summary:    summary,
			Status:     statusNone,
			Open:       open,
			Expandable: true,
			OnToggle:   func() { workflowToggle(m.ThinkingExpanded, key, false) },
		}, disclosureStyle{FG: muted, Muted: muted, Accent: muted, Hover: muted.Alpha(.12), Size: 12})
		if !open {
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Padding(2, 0, 2, 22).Gap(2).Children(func() {
			if len(u.Routes) > 0 {
				usageDetailLine(c, muted, strings.Join(u.Routes, ", "), "")
			}
			usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageUncached), formatTokensCompact(u.Input))
			if u.CacheRead != nil {
				usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageCacheRead), formatTokensCompact(*u.CacheRead))
			}
			if u.CacheWrite != nil {
				usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageCacheWrit), formatTokensCompact(*u.CacheWrite))
			}
			output := formatTokensCompact(u.Output)
			if u.Reasoning != nil {
				output += " · " + i18n.T(i18n.KeyChatUsageReasoning) + " " + formatTokensCompact(*u.Reasoning)
			}
			usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageOutput), output)
			usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageTotal), formatTokensCompact(u.Total))
		})
	})
}

// usageDetailLine is one label/value pair of the usage body (the Original's
// dt/dd pair), kept on one line so the numbers align right.
func usageDetailLine(c *ui.Context, muted ui.Color, label, value string) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(muted)
		if value != "" {
			ui.Text(c, value).FontSize(11).TextColor(muted).Grow(1).TextAlign(ui.End)
		} else {
			ui.Box(c).Grow(1)
		}
	})
}

// formatTokensCompact renders 5_234_567 as "5.2M", matching the Original
// token-format ladder instead of printing raw float digits.
func formatTokensCompact(value float64) string {
	switch {
	case value >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", value/1_000_000_000)
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", value/1_000_000)
	case value >= 10_000:
		return fmt.Sprintf("%.1fK", value/1_000)
	case value >= 1_000:
		return fmt.Sprintf("%.2fK", value/1_000)
	default:
		return fmt.Sprintf("%.0f", value)
	}
}

// PerformanceUsageView is the composer status line's decode/cache summary.
func PerformanceUsageView(c *ui.Context, snapshot session.ViewState, mode string, muted ui.Color) {
	memo := ui.Local(c.Root(), "chat-performance-history-derived", func() session.HistoryDerivedMemo { return session.HistoryDerivedMemo{} })
	tokens, ms, steps := memo.RecordedPerformance(snapshot)
	var labels []string
	if ms > 0 {
		labels = append(labels, fmt.Sprintf(i18n.T(i18n.KeyChatStatusSpeed), tokens*1000/ms))
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
		labels = append(labels, fmt.Sprintf(i18n.T(i18n.KeyChatUsageCachePct), 100*hit/prompt))
	}
	if len(labels) == 0 {
		return
	}
	key := "performance:" + snapshot.PlanHistoryBinding + ":" + snapshot.SelectedID
	row := ui.Column(c).Key(key).FillWidth().Padding(4, 8).Gap(4)
	open := ui.Local(row, "open", func() bool { return false })
	row.Children(func() {
		summary := strings.Join(labels, " · ")
		if mode == "compact" {
			ui.Text(c, summary).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			return
		}
		paintDisclosureRow(c, disclosureRow{
			Key:        key + ":row",
			Title:      summary,
			Open:       *open,
			Expandable: true,
			OnToggle:   func() { *open = !*open },
		}, disclosureStyle{FG: muted, Muted: muted, Accent: muted, Hover: muted.Alpha(.12), Size: 11})
		if !*open {
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Padding(2, 0, 2, 20).Gap(2).Children(func() {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatStatusLoaded), len(snapshot.TurnUsage), steps)).FontSize(11).TextColor(muted)
			if ms > 0 {
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatStatusTimed), tokens, ms)).FontSize(11).TextColor(muted)
			}
			if prompt > 0 {
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatStatusInput), prompt, hit)).FontSize(11).TextColor(muted)
			}
		})
	})
}
