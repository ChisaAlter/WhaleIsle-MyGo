package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	usagestore "whaleisle/internal/store/usage"
	fonttheme "whaleisle/internal/ui/theme"
)

func projectRankingView(c *ui.Context, m Model, st usagestore.StatsView, fg, muted, accent ui.Color) {
	r := m.Store.ProjectCardRanking()
	b := m.Store.Billing()
	seat := ui.Column(c.Key("usage-project-ranking-state/" + m.Store.StatisticsOwner()))
	open := ui.Local(seat, "more", func() bool { return false })
	if !st.Loading && !r.Started && !r.Loading && r.Err == "" {
		m.Store.LoadProjectCard(context.Background(), r.Sort)
	}
	usageCard(c, "usage-project-ranking", usageText("项目排行（会话工作目录）", "Project ranking (session working directories)"), fg, func() {
		value := r.Sort
		rankingSort(c, &value, r.Loading, func() { m.Store.LoadProjectCard(context.Background(), value) })
		projectRankRows(c, r.Rows[:min(10, len(r.Rows))], b, fg, muted)
		if r.Err != "" {
			rankMore(c, r.Loading, r.Started, r.HasMore, r.Err, func() { m.Store.LoadProjectCard(context.Background(), r.Sort) }, muted)
		}
		if ui.Button(c, usageText("更多", "More")).AlignSelf(ui.Start).BorderWidth(0).Background(ui.Transparent).Disabled(len(r.Rows) == 0).Clicked() {
			*open = true
			m.Store.ResetProjectRanking(context.Background(), "tokens")
		}
	})
	r = m.Store.ProjectRanking()
	rankDialog(c, open, usageText("项目排行", "Project ranking"), func() {
		value := r.Sort
		rankingSort(c, &value, r.Loading, func() { m.Store.ResetProjectRanking(context.Background(), value) })
		projectRankRows(c, r.Rows, b, fg, muted)
		rankMore(c, r.Loading, r.Started, r.HasMore, r.Err, func() { m.Store.LoadMoreProjects(context.Background()) }, muted)
	})
}
func projectRankRows(c *ui.Context, rows []usagestore.ProjectUsage, b usagestore.BillingView, fg, muted ui.Color) {
	rankTableHeader(c, false, muted)
	for i, row := range rows {
		name := row.Name
		path := usageText("无工作目录", "No working directory")
		if row.Project != nil {
			path = *row.Project
		}
		if name == "" {
			name = path
		}
		price := "—"
		if cents, ok := usagestore.ProjectEstimate(row, b); ok {
			price = fmt.Sprintf("¥%.2f", cents/100)
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(6, 0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprint(i+1)).Width(22).TextColor(muted)
			ui.Text(c, name).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(path).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
			ui.Text(c, price).Width(62).TextAlign(ui.End).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Text(c, formatCompactTokens(row.Totals.Total)).Width(62).TextAlign(ui.End).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		})
	}
	if len(rows) == 0 {
		ui.Text(c, usageText("暂无项目用量。", "No project usage.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
}
