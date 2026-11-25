package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
func TurnUsageView(c *ui.Context, m Model, msg session.Message, _ ui.Color) {
	if m.Preferences.PerformanceUsage == "compact" {
		return
	}
	u, ok := m.Snapshot.TurnUsage[msg.Seq]
	if !ok {
		return
	}
	t := design.For(c)
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
		trigger := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonSM,
			Label:   i18n.T(i18n.KeyChatUsageTitle),
			Content: func() {
				ui.Box(c).Absolute().Fill().PassThrough().Label(summaryLabel)
				if summaryAlt != "" && summaryAlt != summaryLabel {
					ui.Box(c).Absolute().Fill().PassThrough().Label(summaryAlt)
				}
				ui.Icon(c, usageDatabaseIcon).Size(15+delta, 15+delta)
				w, _ := c.Size()
				if w > 480 {
					ui.Text(c, label).FontSize(12 + delta).FixedLineHeight(24 + delta).SingleLine().MinWidth(0)
				}
			},
		}).Tooltip(summaryLabel).Description(summaryLabel).
			Height(28+delta).MinWidth(0).Padding(0, design.Sp(2)).Gap(design.Sp(1)).Radius(design.RadiusMD).
			TextColor(t.MutedForeground).Expanded(*open)
		if trigger.Clicked() {
			*open = !*open
		}
		w, _ := c.Size()
		if pop := coss.Popover(c, trigger, open, coss.PopoverOptions{Side: coss.SideTop, Align: coss.AlignStart}, func() {
			ui.Column(c).FillWidth().Gap(design.Sp(1.5)).Children(func() {
				ui.Row(c).FillWidth().Gap(design.Sp(4)).Children(func() {
					ui.Icon(c, usageDatabaseIcon).Size(design.IconXS, design.IconXS).TextColor(t.Foreground)
					ui.Text(c, i18n.T(i18n.KeyChatUsageTitle)).FontSize(design.TextXS.Size).FontWeight(design.WeightMedium).TextColor(t.PopoverForeground).Grow(1)
					ui.Text(c, formatExactTokens(u.Total)).FontSize(design.TextXS.Size).FontWeight(design.WeightMedium).TextColor(t.PopoverForeground)
				})
				coss.Separator(c, coss.Horizontal).Margin(design.Sp(0.5), 0, design.Sp(1), 0)
				// The labeled grid mirrors the Original's dl[data-turn-usage-details]
				// (padding 10px 16px 12px 12px, row gap 6) for comparison.
				ui.Column(c).Label("Turn usage details").FillWidth().MinWidth(0).Margin(design.Sp(1), 0, 0, design.Sp(5.5)).Padding(design.Sp(2.5), design.Sp(4), design.Sp(3), design.Sp(3)).Gap(design.Sp(1.5)).Children(func() {
					if len(u.Routes) > 0 {
						usageDetailLine(c, t.MutedForeground, referenceCopy("Model", "模型"), strings.Join(u.Routes, ", "))
					}
					if u.CacheRead != nil && u.Total > u.Output {
						usageDetailLine(c, t.MutedForeground, referenceCopy("Cache hit", "缓存命中"), fmt.Sprintf("%.1f%%", 100**u.CacheRead/(u.Total-u.Output)))
					}
					usageDetailLine(c, t.MutedForeground, i18n.T(i18n.KeyChatUsageUncached), formatExactTokens(u.Input))
					if u.CacheRead != nil {
						usageDetailLine(c, t.MutedForeground, i18n.T(i18n.KeyChatUsageCacheRead), formatExactTokens(*u.CacheRead))
					}
					if u.CacheWrite != nil {
						usageDetailLine(c, t.MutedForeground, i18n.T(i18n.KeyChatUsageCacheWrit), formatExactTokens(*u.CacheWrite))
					}
					output := formatExactTokens(u.Output)
					if u.Reasoning != nil {
						output += " · " + fmt.Sprintf(i18n.T(i18n.KeyChatUsageReasoning), formatExactTokens(*u.Reasoning))
					}
					usageDetailLine(c, t.MutedForeground, i18n.T(i18n.KeyChatUsageOutput), output)
				})
			})
		}); pop.Valid() {
			pop.Role(ui.RoleDialog).Radius(design.RadiusXL2).
				MinWidth(min(float32(300), max(float32(0), w-24))).
				MaxWidth(min(float32(440), max(float32(0), w-24))).
				MaxWidthPercent(96)
		}
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
func usageDetailLine(c *ui.Context, _ ui.Color, label, value string) {
	t := design.For(c)
	ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
		// The Original's details grid column is minmax(76px, auto).
		ui.Text(c, label).FontSize(design.TextXS.Size).FixedLineHeight(18).TextColor(t.MutedForeground).MinWidth(76)
		if value != "" {
			ui.Text(c, value).FontSize(design.TextXS.Size).FixedLineHeight(18).TextColor(t.PopoverForeground).Grow(1).MinWidth(0).TextAlign(ui.End)
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
func PerformanceUsageView(c *ui.Context, snapshot session.ViewState, mode string, _ ui.Color) {
	t := design.For(c)
	muted := t.MutedForeground
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
	row := ui.Column(c).Key(key).FillWidth().Padding(design.Sp(1), design.Sp(2)).Gap(design.Sp(1))
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
		}, disclosureStyle{FG: muted, Muted: muted, Accent: muted, Hover: t.Accent, Size: 11})
		if !*open {
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Padding(design.Sp(0.5), 0, design.Sp(0.5), design.Sp(5)).Gap(design.Sp(0.5)).Children(func() {
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
