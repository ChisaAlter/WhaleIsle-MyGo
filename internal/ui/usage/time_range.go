package usage

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

func timeRangeView(c *ui.Context, m Model) {
	st := m.Store.Stats()
	pagekit.Strong(c, usageText("日期范围（UTC，含首尾）", "Date range (UTC, inclusive)")).FillWidth()
	ui.Row(c).FillWidth().Gap(design.Sp(2)).Wrap().AlignItems(ui.Center).Children(func() {
		start, end := st.RangeStart, st.RangeEnd
		startIn := coss.Input(c, &start, coss.InputOptions{
			Size:        coss.InputSM,
			Placeholder: "YYYY-MM-DD",
			Label:       usageText("开始日期", "Start date"),
		}).Width(145)
		endIn := coss.Input(c, &end, coss.InputOptions{
			Size:        coss.InputSM,
			Placeholder: "YYYY-MM-DD",
			Label:       usageText("结束日期", "End date"),
		}).Width(145)
		if startIn.Changed() || endIn.Changed() {
			m.Store.SetStatsRangeDraft(start, end)
		}
		if coss.Button(c, usageText("应用", "Apply"), coss.ButtonOptions{
			Variant: coss.ButtonDefault,
			Size:    coss.ButtonSM,
		}).Clicked() {
			m.Store.ApplyStatsRangeDraft()
		}
		if coss.Button(c, usageText("清除", "Clear"), coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonSM,
		}).Clicked() {
			m.Store.ClearStatsDateRange()
		}
	})
	if st.RangeError != nil {
		pagekit.FaultSentence(c, "", st.RangeError.SentenceText(), "", nil)
	}
	if st.Range.Active() {
		selected := st.Selection()
		pagekit.Body(c, fmt.Sprintf(usageText("%s → %s UTC · %.0f tokens · %d 天", "%s → %s UTC · %.0f tokens · %d days"), st.Range.Start, st.Range.End, selected.Totals.Total, len(selected.Days))).FillWidth()
		pagekit.Caption(c, usageText("选区仅改变热图汇总；排行和导出保持原统计范围。", "This selection changes only the calendar summary; rankings and exports keep their original scope.")).FillWidth()
	}
	if st.Overview == nil || m.Embedded {
		return
	}
	pagekit.Caption(c, usageText("点击日期选一天；“扩展”将该日加入区间。仅可选择已返回的日期。", "Pick a day, then Extend to include another date. Only returned dates are available.")).FillWidth()
	coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
		ui.Column(c).FillWidth().Gap(design.Sp(1)).Children(func() {
			days := st.Overview.Days
			for offset := 0; offset < len(days); offset += 4 {
				ui.Row(c).Gap(design.Sp(1)).Children(func() {
					for _, d := range days[offset:min(offset+4, len(days))] {
						day := d
						label := fmt.Sprintf("%s · %.0f", day.Date, day.Total)
						if st.Range.Active() && day.Date >= st.Range.Start && day.Date <= st.Range.End {
							label = "✓ " + label
						}
						if coss.Button(c, label, coss.ButtonOptions{
							Variant: coss.ButtonOutline,
							Size:    coss.ButtonXS,
						}).Clicked() {
							m.Store.SelectStatsDay(day.Date, false)
						}
						if st.Range.Active() && coss.Button(c, usageText("扩展", "Extend"), coss.ButtonOptions{
							Variant: coss.ButtonGhost,
							Size:    coss.ButtonXS,
						}).Clicked() {
							m.Store.SelectStatsDay(day.Date, true)
						}
					}
				})
			}
		})
	}).FillWidth().Height(140)
}
