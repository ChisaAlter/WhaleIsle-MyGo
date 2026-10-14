package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	usagestore "whaleisle/internal/store/usage"
)

func rankingView(c *ui.Context, m Model, st usagestore.StatsView, fg, muted, accent ui.Color) {
	models := st.Overview.AllTime.ByModel
	title := usageText("按模型（全部历史）", "Models (all time)")
	if st.Range.Active() {
		models = st.Selection().Models
		title = usageText("按模型（选定日期）", "Models (selected dates)")
	}
	ui.Text(c, title).Bold().TextColor(fg)
	limit := min(len(models), m.Store.ModelRankLimit())
	for _, row := range models[:limit] {
		ui.Text(c, fmt.Sprintf("%s · %s · %.0f tokens", row.Model, row.Provider, row.Total)).FontSize(12).TextColor(fg)
	}
	if len(models) == 0 {
		ui.Text(c, usageText("此范围暂无模型用量。", "No model usage in this range.")).TextColor(muted)
	} else {
		ui.Text(c, fmt.Sprintf(usageText("已显示 %d / %d 个模型", "Showing %d / %d models"), limit, len(models))).FontSize(11).TextColor(muted)
	}
	if limit < len(models) && ui.Button(c, usageText("显示更多模型", "Show more models")).Clicked() {
		m.Store.ShowMoreModels()
	}
	if st.Range.Active() {
		ui.Text(c, usageText("会话排行仅支持全部历史；清除日期选择后可查看和导出。", "Session ranking is all-time only; clear the date selection to view and export it.")).TextColor(muted)
		return
	}
	ui.Text(c, usageText("分页为全部历史；会话导出仍为概览 Top-N。", "Pages cover all time; session export remains the overview Top-N.")).FontSize(11).TextColor(muted)
	rank := m.Store.SessionRanking()
	rows := st.Overview.TopSessions
	title = usageText("Top 会话（全部历史）", "Top sessions (all time)")
	if rank.Started {
		rows = rank.Rows
		title = usageText("会话排行（全部历史）", "Session ranking (all time)")
	}
	ui.Text(c, title).Bold().TextColor(fg)
	for i, row := range rows {
		row := row
		name := row.Title
		if name == "" {
			name = row.ID
		}
		ui.Row(c).Gap(6).Children(func() {
			ui.Text(c, fmt.Sprintf("%d. %s · %.0f tokens", i+1, name, row.Totals.Total)).FontSize(12).TextColor(fg).Grow(1)
			if m.OnOpenSession != nil && ui.Button(c, usageText("打开", "Open")).Clicked() {
				m.OnOpenSession(row.ID)
			}
		})
	}
	if rank.Err != "" {
		ui.Text(c, usageText("加载排行失败：", "Could not load ranking: ")+rank.Err).TextColor(accent)
	}
	if rank.Loading {
		ui.Text(c, usageText("正在加载排行…", "Loading ranking…")).TextColor(muted)
		return
	}
	if rank.Started && len(rows) == 0 {
		ui.Text(c, usageText("暂无会话用量。", "No session usage.")).TextColor(muted)
	}
	if rank.Started && !rank.HasMore && rank.Err == "" {
		ui.Text(c, usageText("已显示统计服务返回的全部会话。", "All sessions returned by the statistics service are shown.")).FontSize(11).TextColor(muted)
		return
	}
	label := usageText("显示更多会话", "Show more sessions")
	if rank.Err != "" {
		label = usageText("重试加载排行", "Retry ranking")
	}
	if !st.Loading && ui.Button(c, label).Clicked() {
		m.Store.LoadMoreSessions(context.Background())
	}
}
