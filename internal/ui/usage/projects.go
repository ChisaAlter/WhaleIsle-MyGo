package usage

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	usagestore "whaleisle/internal/store/usage"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func projectRankingView(c *ui.Context, m Model, st usagestore.StatsView, fg, muted, accent ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Text(c, usageText("项目排行（会话工作目录）", "Project ranking (session working directories)")).Bold().TextColor(fg)
	if st.Range.Active() {
		ui.Text(c, usageText("项目排行仅支持全部历史；清除日期选择后查看。", "Project ranking is all-time only; clear the date selection to view it.")).TextColor(muted)
		return
	}
	r := m.Store.ProjectRanking()
	b := m.Store.Billing()
	ui.Text(c, usageText("按服务端真实工作目录聚合，不对应 Project 任务 ID；费用为已保存价格的估算。", "Host aggregation by working directory, not Project task IDs; costs estimate saved prices.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	ui.Row(c).Gap(6).Children(func() {
		for _, sort := range []string{"tokens", "cost"} {
			sort := sort
			label := usageText("按 Token 排序", "Sort by tokens")
			if sort == "cost" {
				label = usageText("按估算费用排序", "Sort by estimated cost")
			}
			if r.Sort == sort {
				label = "● " + label
			}
			if ui.Button(c, label).Disabled(st.Loading || r.Loading).Clicked() {
				m.Store.ResetProjectRanking(context.Background(), sort)
			}
		}
		if widgets.Button(c, colors, usageText("刷新排行", "Refresh ranking"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(st.Loading || r.Loading).Clicked() {
			m.Store.ResetProjectRanking(context.Background(), r.Sort)
		}
	})
	for i, row := range r.Rows {
		name := row.Name
		path := usageText("无工作目录", "No working directory")
		if row.Project != nil {
			path = *row.Project
		}
		if name == "" {
			name = path
		}
		label := usageText("费用未完整计价", "Cost not fully priced")
		if cents, ok := usagestore.ProjectEstimate(row, b); ok {
			label = fmt.Sprintf("≈ CNY %.2f", cents/100)
		}
		ui.Text(c, fmt.Sprintf("%d. %s · %.0f tokens · %s", i+1, name, row.Totals.Total, label)).TextColor(fg).FontSize(fonttheme.InterfaceSize(12))
		ui.Text(c, path).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if r.Err != "" {
		ui.Text(c, usageText("项目排行失败，保留已加载页：", "Project ranking failed; loaded pages retained: ")+r.Err).TextColor(accent)
	}
	if r.Loading {
		ui.Text(c, usageText("正在加载项目排行…", "Loading project ranking…")).TextColor(muted)
		return
	}
	if r.Started && len(r.Rows) == 0 {
		ui.Text(c, usageText("暂无项目用量。", "No project usage.")).TextColor(muted)
	}
	if r.Started && !r.HasMore && r.Err == "" {
		ui.Text(c, usageText("全部项目已显示。", "All projects shown.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		return
	}
	label := usageText("加载项目排行", "Load project ranking")
	if r.Started {
		label = usageText("更多项目", "More projects")
	}
	if r.Err != "" {
		label = usageText("重试本页", "Retry this page")
	}
	if ui.Button(c, label).Disabled(st.Loading).Clicked() {
		m.Store.LoadMoreProjects(context.Background())
	}
}
