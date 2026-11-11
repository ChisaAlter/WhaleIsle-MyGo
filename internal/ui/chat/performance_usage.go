package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
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

// TurnUsageView is the compact action pill and its anchored exact-accounting
// dialog. Details never expand the transcript's row height.
func TurnUsageView(c *ui.Context, m Model, msg session.Message, muted ui.Color) {
	if m.Preferences.PerformanceUsage == "compact" {
		return
	}
	u, ok := m.Snapshot.TurnUsage[msg.Seq]
	if !ok {
		return
	}
	key := fmt.Sprintf("turn-usage:%s:%s:%.0f", m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, msg.Seq)
	root := ui.Row(c.Key(key)).MinWidth(0)
	open := ui.Local(root, "open", func() bool { return false })
	delta := theme.CurrentPreferences().Chat.Size - 14
	root.Children(func() {
		totalCompact := formatTokensCompact(u.Total)
		label := fmt.Sprintf("%s %s", totalCompact, i18n.T(i18n.KeyChatUsageTokens))
		summaryLabel := i18n.ChatParityf(i18n.KeyParityChatTurnUsageConsumed, "total", totalCompact)
		summaryAlt := ""
		if u.CacheRead != nil && u.Total > u.Output && *u.CacheRead > 0 {
			pct := fmt.Sprintf("%.0f", 100**u.CacheRead/(u.Total-u.Output))
			summaryLabel = i18n.ChatParityf(i18n.KeyParityChatTurnUsageSummaryWithCache, "total", totalCompact, "percent", pct)
			summaryAlt = i18n.ChatParityf(i18n.KeyParityChatTurnUsageSummaryWithCacheV2, "total", totalCompact, "percent", pct)
		}
		trigger := ui.Button(c, "").Label(i18n.T(i18n.KeyChatUsageTitle)).Tooltip(summaryLabel).Description(summaryLabel).Height(28+delta).MinWidth(0).Padding(0, 8).Gap(4).Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Expanded(*open)
		trigger.Children(func() {
			ui.Box(c).Fill().PassThrough().Label(summaryLabel)
			if summaryAlt != "" && summaryAlt != summaryLabel {
				ui.Box(c).Fill().PassThrough().Label(summaryAlt)
			}
			ui.Icon(c, usageDatabaseIcon).Size(15+delta, 15+delta)
			w, _ := c.Size()
			if w > 480 {
				ui.Text(c, label).FontSize(12 + delta).FixedLineHeight(24 + delta).SingleLine().MinWidth(0)
			}
		})
		if trigger.Clicked() {
			*open = !*open
		}
		ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
			w, _ := c.Size()
			chrome.PopoverReveal(c, panel.Role(ui.RoleDialog).AttachTo(trigger, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 8, 0).Width(min(float32(440), max(float32(0), w-24))).MinWidth(min(float32(300), max(float32(0), w-24))).MaxWidthPercent(96).Padding(16).Gap(6).Radius(16).Background(c.Theme().Surface).Border(.5, c.Theme().Border)).Children(func() {
				ui.Row(c).FillWidth().Gap(16).Children(func() {
					ui.Icon(c, usageDatabaseIcon).Size(14, 14).TextColor(c.Theme().Text)
					ui.Text(c, i18n.T(i18n.KeyChatUsageTitle)).FontSize(12).FontWeight(500).Grow(1)
					ui.Text(c, formatExactTokens(u.Total)).FontSize(12).FontWeight(500)
				})
				ui.Box(c).FillWidth().Height(.5).Background(c.Theme().Border).Margin(2, 0, 4, 0)
				if len(u.Routes) > 0 {
					usageDetailLine(c, muted, referenceCopy("Model", "模型"), strings.Join(u.Routes, ", "))
				}
				if u.CacheRead != nil && u.Total > u.Output {
					usageDetailLine(c, muted, referenceCopy("Cache hit", "缓存命中"), fmt.Sprintf("%.1f%%", 100**u.CacheRead/(u.Total-u.Output)))
				}
				usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageUncached), formatExactTokens(u.Input))
				if u.CacheRead != nil {
					usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageCacheRead), formatExactTokens(*u.CacheRead))
				}
				if u.CacheWrite != nil {
					usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageCacheWrit), formatExactTokens(*u.CacheWrite))
				}
				output := formatExactTokens(u.Output)
				if u.Reasoning != nil {
					output += " · " + fmt.Sprintf(i18n.T(i18n.KeyChatUsageReasoning), formatExactTokens(*u.Reasoning))
				}
				usageDetailLine(c, muted, i18n.T(i18n.KeyChatUsageOutput), output)
			})
		})
	})
}

var usageDatabaseIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><ellipse cx="8" cy="3.5" rx="5.5" ry="2"/><path d="M2.5 3.5v9c0 1.1 2.5 2 5.5 2s5.5-.9 5.5-2v-9M2.5 8c0 1.1 2.5 2 5.5 2s5.5-.9 5.5-2"/></svg>`))

func formatExactTokens(value float64) string {
	raw := fmt.Sprintf("%.0f", value)
	for i := len(raw) - 3; i > 0; i -= 3 {
		if raw[i-1] != '-' {
			raw = raw[:i] + "," + raw[i:]
		}
	}
	return raw
}

// usageDetailLine is one label/value pair of the usage body (the Original's
// dt/dd pair), kept on one line so the numbers align right.
func usageDetailLine(c *ui.Context, muted ui.Color, label, value string) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(12).FixedLineHeight(18).TextColor(muted)
		if value != "" {
			ui.Text(c, value).FontSize(12).FixedLineHeight(18).TextColor(c.Theme().Text).Grow(1).TextAlign(ui.End)
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
