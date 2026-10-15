package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/ui/i18n"
)

func usageText(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}
func statsView(c *ui.Context, m Model, fg, muted, accent, field ui.Color) {
	pricingView(c, m, fg, muted, accent)
	repairView(c, m, fg, muted, accent)
	st := m.Store.Stats()
	ui.Text(c, usageText("Token 用量统计", "Token usage statistics")).Bold().TextColor(fg)
	if st.Loading {
		ui.Text(c, usageText("正在读取用量…", "Loading usage…")).TextColor(muted)
	}
	if st.Err != "" {
		ui.Text(c, st.Err).TextColor(accent)
	}
	o := st.Overview
	if o == nil {
		ui.Text(c, usageText("尚无用量数据。刷新以读取内置统计服务。", "No usage data. Refresh to read the built-in statistics service.")).TextColor(muted)
		return
	}
	ui.Text(c, usageText("全部历史", "All time")+fmt.Sprintf(": %.0f tokens · %d sessions", o.AllTime.Totals.Total, o.AllTime.SessionCount)).TextColor(fg)
	t := o.AllTime.Totals
	ui.Text(c, fmt.Sprintf("Input %.0f · Output %.0f · Cache read %.0f · Cache write %.0f", t.Input, t.Output, t.CacheRead, t.CacheWrite)).FontSize(12).TextColor(muted)
	ui.Text(c, usageText("近30天", "Recent 30 days")+fmt.Sprintf(": %.0f tokens · %d sessions", o.Totals.Total, o.SessionCount)).TextColor(fg)
	ui.Text(c, totalsText(o.Totals)).FontSize(12).TextColor(muted)
	stamp := time.UnixMilli(o.UpdatedAt).UTC().Format("2006-01-02 15:04:05 UTC")
	ui.Text(c, usageText("最近成功更新：", "Last successful update: ")+stamp).FontSize(11).TextColor(muted)
	if st.Cached {
		ui.Text(c, usageText("已恢复上次启动的缓存；时间仍为上次成功更新。", "Restored cached overview. Timestamp is the last successful update.")).FontSize(11).TextColor(muted)
	}
	if st.Cached || o.Stale || st.Err != "" {
		ui.Text(c, usageText("正在显示上次结果，可能已过期。", "Showing previous results; they may be stale.")).TextColor(accent)
	}
	cv := o.Coverage
	ui.Text(c, fmt.Sprintf("Coverage: %d/%d · Failed %d · Pending %d · %s · UTC", cv.SessionsOk, cv.SessionsTotal, cv.SessionsFailed, cv.SessionsPending, cv.Mode)).FontSize(11).TextColor(muted)
	ui.Text(c, fmt.Sprintf("Events %d · Retries %d · Compaction %.0f tokens · Main %d · Subagent %d", cv.EventsCounted, cv.Retries, cv.CompactionTokens, cv.UsageSessionsMain, cv.UsageSessionsSubagent)).FontSize(11).TextColor(muted)
	if len(cv.FailedSessionIDs) > 0 {
		ui.Text(c, usageText("读取失败的会话：", "Sessions that could not be read: ")+strings.Join(cv.FailedSessionIDs, ", ")).FontSize(11).TextColor(accent)
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, n := range []int{7, 14, 30} {
			if ui.Button(c, fmt.Sprintf("%d days", n)).Clicked() {
				m.Store.SelectStatsRecentDays(n)
			}
		}
		if ui.Button(c, usageText("重新扫描", "Rescan")).Clicked() {
			m.Store.RefreshStats(context.Background(), true)
		}
	})
	timeRangeView(c, m, fg, muted, accent)
	statisticsDimensionsView(c, m, st, fg, muted, accent, field)
	if st.Range.Active() {
		ui.Text(c, usageText("JSON / 每日 / 模型导出：当前选定 UTC 日期。费用为 CNY 分；未知留空，部分计价另列覆盖数。", "JSON / daily / model exports: selected UTC dates. Costs are CNY cents; unknown values are blank and partial pricing has coverage counts.")).FontSize(11).TextColor(muted)
	} else {
		ui.Text(c, usageText("模型导出：近30天；每日导出：热力图全部可用 UTC 日期；会话导出：全部历史 Top-N。费用为 CNY 分，计价覆盖数区分部分估算。", "Model export: recent 30 days; daily export: all available heatmap UTC dates; sessions: all-time Top-N. Costs are CNY cents with pricing coverage counts.")).FontSize(11).TextColor(muted)
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, kind := range []string{"json", "days", "models", "sessions"} {
			kind := kind
			if st.Range.Active() && kind == "sessions" {
				continue
			}
			if m.OnExportStats != nil && ui.Button(c, usageText("导出 ", "Export ")+kind).Clicked() {
				m.OnExportStats(kind)
			}
		}
	})
	rankingView(c, m, st, fg, muted, accent)
	projectRankingView(c, m, st, fg, muted, accent)

}
