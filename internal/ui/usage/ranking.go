package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	usagestore "whaleisle/internal/store/usage"
	fonttheme "whaleisle/internal/ui/theme"
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
		ui.Text(c, fmt.Sprintf("%s · %s · %.0f tokens", row.Model, row.Provider, row.Total)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if len(models) == 0 {
		ui.Text(c, usageText("此范围暂无模型用量。", "No model usage in this range.")).TextColor(muted)
	} else {
		ui.Text(c, fmt.Sprintf(usageText("已显示 %d / %d 个模型", "Showing %d / %d models"), limit, len(models))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if limit < len(models) && ui.Button(c, usageText("显示更多模型", "Show more models")).Clicked() {
		m.Store.ShowMoreModels()
	}
	if st.Range.Active() {
		ui.Text(c, usageText("会话排行仅支持全部历史；清除日期选择后可查看和导出。", "Session ranking is all-time only; clear the date selection to view and export it.")).TextColor(muted)
		return
	}
	ui.Text(c, usageText("分页为全部历史；会话导出仍为概览 Top-N。", "Pages cover all time; session export remains the overview Top-N.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	rank := m.Store.SessionRanking()
	billing := m.Store.Billing()
	ui.Row(c).Gap(6).Children(func() {
		for _, sort := range []string{"tokens", "cost"} {
			sort := sort
			label := usageText("按 Token 排序", "Sort by tokens")
			if sort == "cost" {
				label = usageText("按估算费用排序", "Sort by estimated cost")
			}
			if rank.Sort == sort {
				label = "● " + label
			}
			if ui.Button(c, label).Disabled(st.Loading || rank.Loading).Clicked() {
				m.Store.ResetSessionRanking(context.Background(), sort)
			}
		}
		if ui.Button(c, usageText("刷新会话排行", "Refresh session ranking")).Disabled(st.Loading || rank.Loading).Clicked() {
			m.Store.ResetSessionRanking(context.Background(), rank.Sort)
		}
	})
	rows := st.Overview.TopSessions
	title = usageText("Top 会话（全部历史）", "Top sessions (all time)")
	if rank.Started || rank.Sort == "cost" {
		rows = rank.Rows
		title = usageText("会话排行（全部历史）", "Session ranking (all time)")
	}
	ui.Text(c, title).Bold().TextColor(fg)
	if rank.Sort == "cost" && !rank.Started && !rank.Loading && rank.Err == "" && !st.Loading {
		m.Store.LoadMoreSessions(context.Background())
	}
	for i, row := range rows {
		row := row
		name := row.Title
		if name == "" {
			name = row.ID
		}
		ui.Row(c).Gap(6).Children(func() {
			cost := usageText("费用未完整计价", "Cost not fully priced")
			if cents, ok := usagestore.SessionEstimate(row, billing); ok {
				cost = fmt.Sprintf("≈ CNY %.2f", cents/100)
			}
			ui.Text(c, fmt.Sprintf("%d. %s · %.0f tokens · %s", i+1, name, row.Totals.Total, cost)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
			if m.OnOpenSession != nil && ui.Button(c, usageText("打开", "Open")).Clicked() {
				m.OnOpenSession(row.ID)
			}
		})
		kind := usageText("主会话", "Main session")
		if row.Depth > 0 {
			kind = usageText("子代理会话", "Subagent session")
		}
		if row.LastActive > 0 {
			kind += " · " + time.UnixMilli(row.LastActive).UTC().Format("2006-01-02 15:04 UTC")
		}
		ui.Text(c, kind).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
		ui.Text(c, usageText("已显示统计服务返回的全部会话。", "All sessions returned by the statistics service are shown.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
