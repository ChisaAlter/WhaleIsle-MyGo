package usage

import (
	"fmt"
	"github.com/egoist/mygo/ui"
)

func timeRangeView(c *ui.Context, m Model, fg, muted, accent ui.Color) {
	st := m.Store.Stats()
	ui.Text(c, usageText("日期范围（UTC，含首尾）", "Date range (UTC, inclusive)")).TextColor(fg)
	ui.Row(c).Gap(6).Children(func() {
		start, end := st.RangeStart, st.RangeEnd
		ui.TextInput(c, &start).Width(145).Placeholder("YYYY-MM-DD").OnChange(func() { m.Store.SetStatsRangeDraft(start, end) })
		ui.TextInput(c, &end).Width(145).Placeholder("YYYY-MM-DD").OnChange(func() { m.Store.SetStatsRangeDraft(start, end) })
		if ui.Button(c, usageText("应用", "Apply")).Clicked() {
			m.Store.ApplyStatsRangeDraft()
		}
		if ui.Button(c, usageText("清除", "Clear")).Clicked() {
			m.Store.ClearStatsDateRange()
		}
	})
	if st.RangeError != "" {
		ui.Text(c, st.RangeError).TextColor(accent)
	}
	if st.Range.Active() {
		selected := st.Selection()
		ui.Text(c, fmt.Sprintf("%s → %s UTC · %.0f tokens · %d days", st.Range.Start, st.Range.End, selected.Totals.Total, len(selected.Days))).TextColor(fg)
		ui.Text(c, usageText("日统计、模型排行和导出使用相同选区；扫描覆盖情况仍适用。", "Daily statistics, model ranking and exports use this selection; scan coverage still applies.")).FontSize(11).TextColor(muted)
	}
	if st.Overview == nil {
		return
	}
	ui.Text(c, usageText("点击日期选一天；“扩展”将该日加入区间。仅可选择已返回的日期。", "Pick a day, then Extend to include another date. Only returned dates are available.")).FontSize(11).TextColor(muted)
	ui.Scroll(c).Height(140).Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			days := st.Overview.Days
			for offset := 0; offset < len(days); offset += 4 {
				ui.Row(c).Gap(4).Children(func() {
					for _, d := range days[offset:min(offset+4, len(days))] {
						day := d
						label := fmt.Sprintf("%s · %.0f", day.Date, day.Total)
						if st.Range.Active() && day.Date >= st.Range.Start && day.Date <= st.Range.End {
							label = "✓ " + label
						}
						if ui.Button(c, label).FontSize(11).Clicked() {
							m.Store.SelectStatsDay(day.Date, false)
						}
						if st.Range.Active() && ui.Button(c, usageText("扩展", "Extend")).FontSize(10).Clicked() {
							m.Store.SelectStatsDay(day.Date, true)
						}
					}
				})
			}
		})
	})
}
