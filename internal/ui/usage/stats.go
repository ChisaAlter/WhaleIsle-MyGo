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
	stamp := time.UnixMilli(o.UpdatedAt).UTC().Format("2006-01-02 15:04:05 UTC")
	ui.Text(c, usageText("最近成功更新：", "Last successful update: ")+stamp).FontSize(11).TextColor(muted)
	if o.Stale || st.Err != "" {
		ui.Text(c, usageText("正在显示上次结果，可能已过期。", "Showing previous results; they may be stale.")).TextColor(accent)
	}
	cv := o.Coverage
	ui.Text(c, fmt.Sprintf("Coverage: %d/%d · Failed %d · Pending %d · %s · UTC", cv.SessionsOk, cv.SessionsTotal, cv.SessionsFailed, cv.SessionsPending, cv.Mode)).FontSize(11).TextColor(muted)
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
	days := o.Days
	if st.Range.Active() {
		days = st.Selection().Days
	}
	if !st.Range.Active() && len(days) > st.Days {
		days = days[len(days)-st.Days:]
	}
	bars := make([]Bar, 0, len(days))
	for _, d := range days {
		label := d.Date
		if len(label) > 5 {
			label = label[5:]
		}
		bars = append(bars, Bar{Label: label, Value: d.Total})
	}
	ui.Text(c, usageText("每日 Token（UTC）", "Daily tokens (UTC)")).TextColor(fg)
	// Split into week-sized rows so every day remains reachable on narrow windows.
	ui.Scroll(c).Height(160).Children(func() {
		ui.Column(c).Gap(8).Children(func() {
			for start := 0; start < len(bars); start += 7 {
				end := min(start+7, len(bars))
				BarChartView(c, bars[start:end], fg, muted, accent, field)
			}
		})
	})
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
