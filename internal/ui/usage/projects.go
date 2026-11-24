package usage

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

func projectRankingView(c *ui.Context, m Model, st usagestore.StatsView) {
	r := m.Store.ProjectCardRanking()
	b := m.Store.Billing()
	seat := ui.Column(c.Key("usage-project-ranking-state/" + m.Store.StatisticsOwner()))
	open := ui.Local(seat, "more", func() bool { return false })
	if !st.Loading && !r.Started && !r.Loading && r.Err == nil {
		m.Store.LoadProjectCard(context.Background(), r.Sort)
	}
	usageCard(c, "usage-project-ranking", usageText("项目排行（会话工作目录）", "Project ranking (session working directories)"), func() {
		value := r.Sort
		rankingSort(c, &value, r.Loading, func() { m.Store.LoadProjectCard(context.Background(), value) })
		projectRankRows(c, r.Rows[:min(10, len(r.Rows))], b)
		if r.Err != nil {
			rankMore(c, r.Loading, r.Started, r.HasMore, r.Err.SentenceText(), func() { m.Store.LoadProjectCard(context.Background(), r.Sort) })
		}
		if coss.Button(c, usageText("更多", "More"), coss.ButtonOptions{
			Variant:  coss.ButtonGhost,
			Size:     coss.ButtonSM,
			Disabled: len(r.Rows) == 0,
		}).AlignSelf(ui.Start).Clicked() {
			*open = true
			m.Store.ResetProjectRanking(context.Background(), "tokens")
		}
	})
	r = m.Store.ProjectRanking()
	rankDialog(c, open, usageText("项目排行", "Project ranking"), func() {
		value := r.Sort
		rankingSort(c, &value, r.Loading, func() { m.Store.ResetProjectRanking(context.Background(), value) })
		projectRankRows(c, r.Rows, b)
		rankMore(c, r.Loading, r.Started, r.HasMore, r.Err.SentenceText(), func() { m.Store.LoadMoreProjects(context.Background()) })
	})
}

func projectRankRows(c *ui.Context, rows []usagestore.ProjectUsage, b usagestore.BillingView) {
	t := design.For(c)
	rankTableHeader(c, false)
	for i, row := range rows {
		if i > 0 {
			coss.Separator(c, coss.Horizontal)
		}
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
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).PaddingY(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
			pagekit.Caption(c, fmt.Sprint(i+1)).Width(22).FontFeatures("tnum")
			ui.Text(c, name).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(path).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.Foreground)
			pagekit.Caption(c, price).Width(62).TextAlign(ui.End).FontFeatures("tnum")
			ui.Text(c, formatCompactTokens(row.Totals.Total)).Width(62).TextAlign(ui.End).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).FontFeatures("tnum").TextColor(t.Foreground)
		})
	}
	if len(rows) == 0 {
		pagekit.Muted(c, usageText("暂无项目用量。", "No project usage.")).FillWidth()
	}
}
