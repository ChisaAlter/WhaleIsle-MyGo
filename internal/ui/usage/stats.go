package usage

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
)

func usageText(zh, en string) string {
	return i18n.Pick(zh, en)
}

// statsView paints the usage dashboard: the KPI cards, the calendar, the daily
// breakdown, the ranking cards, model share and — folded away — diagnostics.
func statsView(c *ui.Context, m Model) {
	st := m.Store.Stats()
	// Management actions remain available when every log failed or no model has
	// recorded usage yet; neither requires a non-empty chart.
	defer repairView(c, m)
	switch {
	case st.Loading && st.Overview == nil:
		pagekit.SkeletonRows(c, 3, i18n.T(i18n.KeySurfacesUsageLoading))
		return
	case st.Err != nil && st.Overview == nil:
		pagekit.FaultSentence(c, i18n.T(i18n.KeySurfacesUsageError),
			st.ErrLead+st.Err.SentenceText()+i18n.T(i18n.KeySurfacesUsageRetryHint),
			usageText("重试", "Retry"),
			func() { m.Store.RefreshStats(context.Background(), true) })
		return
	case st.Overview == nil:
		usageEmpty(c, i18n.T(settingsPass4UsageEmptyTitle), i18n.T(settingsPass4UsageEmptyBody))
		return
	}
	o := st.Overview
	if m.Embedded && o.AllTime.Totals.Total == 0 && o.AllTime.SessionCount == 0 && o.Totals.Total == 0 && o.SessionCount == 0 {
		// Never-used state: the original's dsw-ust-empty card copy.
		usageEmpty(c, i18n.T(settingsPass4UsageEmptyTitle), i18n.T(settingsPass4UsageEmptyBody))
		return
	}
	if st.Err != nil {
		pagekit.FaultSentence(c, "", st.ErrLead+st.Err.SentenceText(), usageText("重试", "Retry"), func() {
			m.Store.RefreshStats(context.Background(), true)
		})
	}

	statisticsDimensionsView(c, m, st)
	usageDiagnostics(c, st)
}

// formatCompactTokens renders a token count the way a status line does: 12.4K,
// 1.2M, 77.7M. A value below a thousand keeps its integer form.
func formatCompactTokens(value float64) string {
	units := []struct {
		limit  float64
		suffix string
	}{{1e9, "B"}, {1e6, "M"}, {1e3, "K"}}
	for _, unit := range units {
		if value < unit.limit {
			continue
		}
		scaled := value / unit.limit
		if scaled >= 100 {
			return fmt.Sprintf("%.0f%s", scaled, unit.suffix)
		}
		return fmt.Sprintf("%.1f%s", scaled, unit.suffix)
	}
	return fmt.Sprintf("%.0f", value)
}

// usageRangeToolbar is the dashboard's recent-day window selector.
func usageRangeToolbar(c *ui.Context, m Model, st usagestore.StatsView) {
	windows := []int{7, 14, 30}
	labels := make([]string, len(windows))
	selected := 0
	for i, d := range windows {
		labels[i] = fmt.Sprintf(i18n.T(i18n.KeySurfacesUsageDays), d)
		if st.Days == d {
			selected = i
		}
	}
	seg := coss.SegmentedControl(c, &selected, labels, coss.SegmentedOptions{
		Size:     coss.SegmentedSM,
		Disabled: st.Loading,
		Label:    usageText("统计窗口", "Time window"),
		Key:      "usage-days-window",
	})
	if seg.Changed() {
		m.Store.SetStatsDays(windows[selected])
	}
}

// usageDiagnostics folds the scan coverage and event counters behind one
// disclosure so the dashboard is not a diagnostics page.
func usageDiagnostics(c *ui.Context, st usagestore.StatsView) {
	o := st.Overview
	anchor := ui.Column(c.Key("usage-diagnostics")).FillWidth().MinWidth(0).Gap(design.Sp(2))
	open := ui.Local(anchor, "open", func() bool { return false })
	anchor.Children(func() {
		t := design.For(c)
		label := i18n.T(i18n.KeySurfacesUsageDiagnostic)
		toggle := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonSM,
			Label:   label,
			Content: func() {
				glyph := coss.IconChevronRight
				if *open {
					glyph = coss.IconChevronDown
				}
				ui.Icon(c, glyph.SVG()).Size(design.IconSM, design.IconSM).Shrink(0).TextColor(t.MutedForeground).PassThrough()
				pagekit.Muted(c, label).Grow(1).MinWidth(0).SingleLine()
			},
		})
		toggle.FillWidth().Justify(ui.Start).Expanded(*open).Tooltip(label)
		if toggle.Clicked() {
			*open = !*open
		}
		if !*open {
			return
		}
		cv := o.Coverage
		pagekit.Caption(c, fmt.Sprintf("%s: %d/%d · %s %d · %s %d · %s",
			i18n.T(i18n.KeySurfacesUsageCoverage), cv.SessionsOk, cv.SessionsTotal,
			usageText("失败", "failed"), cv.SessionsFailed,
			usageText("待处理", "pending"), cv.SessionsPending,
			cv.Timezone)).FillWidth()
		pagekit.Caption(c, fmt.Sprintf("%s %d · %s %d · %s %s · %s %d · %s %d",
			usageText("事件", "events"), cv.EventsCounted,
			usageText("重试", "retries"), cv.Retries,
			usageText("压缩", "compaction"), formatCompactTokens(cv.CompactionTokens),
			usageText("主会话", "main"), cv.UsageSessionsMain,
			usageText("子代理", "subagent"), cv.UsageSessionsSubagent)).FillWidth()
		if len(cv.FailedSessionIDs) > 0 {
			pagekit.Danger(c, usageText("读取失败的会话：", "Sessions that could not be read: ")+strings.Join(cv.FailedSessionIDs, ", ")).FillWidth()
		}
	})
}
