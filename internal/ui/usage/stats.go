package usage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func usageText(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}

// statsView paints the usage dashboard: the range toolbar, the KPI cards, the
// calendar, the model breakdown, the ranking cards and — folded away — the
// diagnostics and export controls that used to lead the page as raw text.
func statsView(c *ui.Context, m Model, fg, muted, accent, field ui.Color) {
	st := m.Store.Stats()
	switch {
	case st.Loading && st.Overview == nil:
		ui.Text(c, i18n.T(i18n.KeySurfacesUsageLoading)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		return
	case st.Err != "" && st.Overview == nil:
		usageEmpty(c, "", st.Err, fg, muted)
		return
	case st.Overview == nil:
		usageEmpty(c, i18n.T(i18n.KeySurfacesUsageEmpty), usageText("刷新以读取内置统计服务。", "Refresh to read the built-in statistics service."), fg, muted)
		return
	}
	o := st.Overview
	if m.Embedded && o.AllTime.Totals.Total == 0 && o.AllTime.SessionCount == 0 && o.Totals.Total == 0 && o.SessionCount == 0 {
		usageEmpty(c, i18n.T(i18n.KeySurfacesUsageEmpty), usageText("开始使用模型后统计会显示在这里。", "Usage appears here after you use a model."), fg, muted)
		return
	}
	if st.Err != "" {
		ui.Text(c, st.Err).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
	}
	usageRangeToolbar(c, m, st, fg, muted, accent)
	statisticsDimensionsView(c, m, st, fg, muted, accent, field)
	rankingView(c, m, st, fg, muted, accent)
	projectRankingView(c, m, st, fg, muted, accent)
	usageDiagnostics(c, m, st, fg, muted, accent)
	pricingView(c, m, fg, muted, accent)
	repairView(c, m, fg, muted, accent)
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

// usageRangeToolbar is the dashboard's tool row: the recent-day choices, a
// rescan action and the date-range editor.
func usageRangeToolbar(c *ui.Context, m Model, st usagestore.StatsView, fg, muted, accent ui.Color) {
	ui.Row(c).FillWidth().Wrap().Gap(4).AlignItems(ui.Center).Children(func() {
		for _, days := range []int{7, 14, 30} {
			window := days
			label := fmt.Sprintf(i18n.T(i18n.KeySurfacesUsageDays), window)
			usageScopeChip(c, m, label, !st.Range.Active() && st.Days == window, st.Loading, func() {
				m.Store.SelectStatsRecentDays(window)
			})
		}
		ui.Box(c).Width(1).Height(16).Shrink(0).Background(c.Theme().Border)
		if ui.Button(c, i18n.T(i18n.KeySurfacesUsageScan)).FontSize(fonttheme.InterfaceSize(12)).Disabled(st.Loading).Clicked() {
			m.Store.RefreshStats(context.Background(), true)
		}
		ui.Box(c).Grow(1)
		if st.Overview.UpdatedAt > 0 {
			stamp := time.UnixMilli(st.Overview.UpdatedAt).UTC().Format("2006-01-02 15:04 UTC")
			ui.Text(c, i18n.T(i18n.KeySurfacesUsageUpdated)+" "+stamp).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
	})
	if st.Cached || st.Overview.Stale {
		ui.Text(c, i18n.T(i18n.KeySurfacesUsageStale)).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	}
	timeRangeView(c, m, fg, muted, accent)
	usageExportRow(c, m, st)
}

// usageExportRow keeps the export actions next to the range they respect: the
// session export needs the all-time scope, so it disappears while a date range
// is active.
func usageExportRow(c *ui.Context, m Model, st usagestore.StatsView) {
	if m.OnExportStats == nil {
		return
	}
	ui.Row(c).FillWidth().Wrap().Gap(4).AlignItems(ui.Center).Children(func() {
		for _, kind := range []struct{ id, zh, en string }{
			{"json", "导出 JSON", "Export JSON"},
			{"days", "导出每日 CSV", "Export daily CSV"},
			{"models", "导出模型 CSV", "Export model CSV"},
			{"sessions", "导出会话 CSV", "Export session CSV"},
		} {
			format := kind
			if format.id == "sessions" && st.Range.Active() {
				continue
			}
			if ui.Button(c, usageText(format.zh, format.en)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
				m.OnExportStats(format.id)
			}
		}
	})
}

// usageDiagnostics folds the scan coverage, the event counters and the export
// actions behind one disclosure so the dashboard is not a diagnostics page.
func usageDiagnostics(c *ui.Context, m Model, st usagestore.StatsView, fg, muted, accent ui.Color) {
	o := st.Overview
	anchor := ui.Column(c.Key("usage-diagnostics")).FillWidth().MinWidth(0).Gap(8)
	open := ui.Local(anchor, "open", func() bool { return false })
	anchor.Children(func() {
		toggle := ui.Button(c, "").Label(i18n.T(i18n.KeySurfacesUsageDiagnostic)).FillWidth().Height(30).Padding(0, 4).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Tooltip(i18n.T(i18n.KeySurfacesUsageDiagnostic))
		toggle.Children(func() {
			if *open {
				ui.Icon(c, usageChevronDownIcon).Size(10, 10).TextColor(muted).PassThrough()
			} else {
				ui.Icon(c, usageChevronRightIcon).Size(10, 10).TextColor(muted).PassThrough()
			}
			ui.Text(c, " "+i18n.T(i18n.KeySurfacesUsageDiagnostic)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Grow(1)
		})
		if toggle.Hovered() {
			toggle.Background(c.Theme().SurfaceHover)
		}
		if toggle.Clicked() {
			*open = !*open
		}
		if !*open {
			return
		}
		cv := o.Coverage
		ui.Text(c, fmt.Sprintf("%s: %d/%d · %s %d · %s %d · %s",
			i18n.T(i18n.KeySurfacesUsageCoverage), cv.SessionsOk, cv.SessionsTotal,
			usageText("失败", "failed"), cv.SessionsFailed,
			usageText("待处理", "pending"), cv.SessionsPending,
			cv.Timezone)).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, fmt.Sprintf("%s %d · %s %d · %s %s · %s %d · %s %d",
			usageText("事件", "events"), cv.EventsCounted,
			usageText("重试", "retries"), cv.Retries,
			usageText("压缩", "compaction"), formatCompactTokens(cv.CompactionTokens),
			usageText("主会话", "main"), cv.UsageSessionsMain,
			usageText("子代理", "subagent"), cv.UsageSessionsSubagent)).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if len(cv.FailedSessionIDs) > 0 {
			ui.Text(c, usageText("读取失败的会话：", "Sessions that could not be read: ")+strings.Join(cv.FailedSessionIDs, ", ")).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
		}
	})
}
