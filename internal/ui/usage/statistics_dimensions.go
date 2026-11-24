package usage

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
	"whaleisle/internal/ui/theme"
)

func totalsText(t usagestore.Totals) string {
	return fmt.Sprintf(usageText("输入 %.0f · 输出 %.0f · 缓存读取 %.0f · 缓存写入 %.0f", "Input %.0f · Output %.0f · Cache read %.0f · Cache write %.0f"), t.Input, t.Output, t.CacheRead, t.CacheWrite)
}

func phaseTotalsText(c *usagestore.PhaseBuckets) string {
	if c == nil || c.Peak == nil || c.OffPeak == nil {
		return ""
	}
	peak := fmt.Sprintf(usageText("高峰 命中 %.0f · 未命中 %.0f · 输出 %.0f", "Peak hit %.0f · miss %.0f · output %.0f"), c.Peak.CacheRead, c.Peak.Input+c.Peak.CacheWrite, c.Peak.Output)
	idle := fmt.Sprintf(usageText("空闲 命中 %.0f · 未命中 %.0f · 输出 %.0f", "Off-peak hit %.0f · miss %.0f · output %.0f"), c.OffPeak.CacheRead, c.OffPeak.Input+c.OffPeak.CacheWrite, c.OffPeak.Output)
	return peak + "\n" + idle
}

func usageCostText(models []usagestore.ModelUsage, settings usagestore.BillingSettings) string {
	peak, idle, priced, unknown := 0.0, 0.0, 0, 0
	for _, row := range models {
		p, i, ok := usagestore.EstimateModelPhases(row, settings)
		if !ok {
			unknown++
			continue
		}
		peak += p
		idle += i
		priced++
	}
	if priced == 0 {
		return usageText("费用未计价", "Cost unpriced")
	}
	return fmt.Sprintf(usageText("≈ CNY %.2f（峰值 %.2f，谷值 %.2f）· 已计价 %d，未计价 %d", "≈ CNY %.2f (peak %.2f, off-peak %.2f) · %d priced, %d unpriced"), (peak+idle)/100, peak/100, idle/100, priced, unknown)
}

func dayModels(day usagestore.DayUsage, providers map[string]string) []usagestore.ModelUsage {
	rows := make([]usagestore.ModelUsage, 0, len(day.Models))
	for name, t := range day.Models {
		rows = append(rows, usagestore.ModelUsage{Model: name, Provider: providers[name], Totals: t, Cost: day.ModelCosts[name]})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total == rows[j].Total {
			return rows[i].Model < rows[j].Model
		}
		return rows[i].Total > rows[j].Total
	})
	return rows
}

func dayTip(day usagestore.DayUsage, providers map[string]string, settings usagestore.BillingSettings) string {
	rows := dayModels(day, providers)
	lines := []string{fmt.Sprintf("%s UTC · %.0f tokens", day.Date, day.Total), usageCostText(rows, settings)}
	for _, row := range rows {
		detail := totalsText(row.Totals)
		if phases := phaseTotalsText(row.Cost); phases != "" {
			detail += "\n" + phases
		}
		lines = append(lines, fmt.Sprintf("%s · %s · %.0f\n%s\n%s", row.Provider, row.Model, row.Total, detail, usageCostText([]usagestore.ModelUsage{row}, settings)))
	}
	return strings.Join(lines, "\n")
}

func effectiveBillingSettings(b usagestore.BillingView) usagestore.BillingSettings {
	if !b.Loaded {
		return usagestore.BillingSettings{PeakValleyEnabled: true}
	}
	return b.Settings
}

func statisticsDimensionsView(c *ui.Context, m Model, st usagestore.StatsView) {
	o := st.Overview
	billing := m.Store.Billing()
	settings := effectiveBillingSettings(billing)
	providers := map[string]string{}
	for _, row := range o.AllTime.ByModel {
		providers[row.Model] = row.Provider
	}
	usageKPIView(c, m, o, o.AllTime.ByModel, billing)
	calendarUsageView(c, m, st, providers, settings)
	days := o.Days
	if len(days) > st.Days {
		days = days[len(days)-st.Days:]
	}
	usageCard(c, "usage-daily-models", usageText("每日模型堆叠（近期）", "Daily model breakdown (recent)"), func() {
		usageRangeToolbar(c, m, st)
		stackedUsageView(c, days, o.ByModel, providers, settings)
	})
	rankingView(c, m, st)
	projectRankingView(c, m, st)
	if len(o.Providers) > 0 && !(len(o.Providers) == 1 && (o.Providers[0].ID == "unknown" || o.Providers[0].Totals.Total <= 0)) {
		usageCard(c, "usage-providers", usageText("服务商分项用量", "Provider token breakdown"), func() {
			maximum := 1.0
			for _, p := range o.Providers {
				maximum = math.Max(maximum, p.Totals.Total)
			}
			for _, p := range o.Providers {
				name := p.Name
				if name == "" {
					name = p.ID
				}
				coss.Progress(c, p.Totals.Total/maximum, coss.ProgressOptions{
					Label: name,
					Value: formatCompactTokens(p.Totals.Total),
				})
			}
		})
	}
	usageCard(c, "usage-model-share", usageText("模型占比（全部历史）", "Model share (all time)"), func() {
		modelShareView(c, o.AllTime.ByModel, o.AllTime.Totals.Total, settings)
	})
}

// Calendar cells use one set of quartiles over all returned days, including
// when a date selection is active. Gaps are blank rather than zero-use days.
func calendarUsageView(c *ui.Context, m Model, st usagestore.StatsView, providers map[string]string, settings usagestore.BillingSettings) {
	days := append([]usagestore.DayUsage(nil), st.Overview.Days...)
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	if len(days) == 0 {
		return
	}
	t := design.For(c)
	positives := []float64{}
	lookup := map[string]usagestore.DayUsage{}
	for _, d := range days {
		lookup[d.Date] = d
		if d.Total > 0 {
			positives = append(positives, d.Total)
		}
	}
	sort.Float64s(positives)
	levels := []float64{0, 0, 0}
	if len(positives) > 0 {
		for i := range levels {
			levels[i] = positives[(len(positives)-1)*(i+1)/4]
		}
	}
	selected := st.Selection()
	active := 0
	for _, d := range selected.Days {
		if d.Total > 0 {
			active++
		}
	}
	summary := fmt.Sprintf(usageText("%.0f tokens · %d 天有用量，共 %d 天 · %s", "%.0f tokens · %d active of %d days · %s"),
		selected.Totals.Total, active, len(selected.Days), usageCostText(selected.Models, settings))
	var clearAction func()
	if st.Range.Active() {
		clearAction = func() {
			if coss.Button(c, usageText("清除选区", "Clear selection"), coss.ButtonOptions{
				Variant: coss.ButtonGhost,
				Size:    coss.ButtonSM,
			}).Clicked() {
				m.Store.ClearStatsDateRange()
			}
		}
	}
	seat := ui.Column(c.Key("usage-calendar")).FillWidth().MinWidth(0)
	detail := ui.Local(seat, "detail-date", func() string { return "" })
	focusDay := ui.Local(seat, "focus-date", func() string { return "" })
	seat.Children(func() {
		coss.Card(c, coss.CardOptions{
			Key:         "usage-calendar-card",
			Title:       usageText("用量日历（UTC）", "Usage calendar (UTC)"),
			Description: summary,
			Action:      clearAction,
		}, func() {
			buttons := map[string]ui.Element{}
			focusTarget := ""
			start, _ := time.Parse("2006-01-02", days[0].Date)
			end, _ := time.Parse("2006-01-02", days[len(days)-1].Date)
			start = start.AddDate(0, 0, -(int(start.Weekday())+6)%7)
			months := calendarMonthLabels(start, end, lookup)
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Height(124).FocusGroup(ui.Horizontal | ui.Vertical).Children(func() {
				ui.Row(c).Gap(design.Sp(1.5)).Children(func() {
					ui.Column(c).Width(20).Gap(design.Sp(0.75)).Children(func() {
						ui.Box(c).Height(17)
						for _, label := range []string{usageText("一", "Mon"), "", usageText("三", "Wed"), "", usageText("五", "Fri"), "", ""} {
							ui.Text(c, label).Height(12).FixedLineHeight(12).FontSize(theme.InterfaceSize(10)).TextColor(t.MutedForeground).NoWrap()
						}
					})
					ui.Row(c).Gap(design.Sp(0.75)).Children(func() {
						for week := start; !week.After(end); week = week.AddDate(0, 0, 7) {
							ui.Column(c).Width(12).Gap(design.Sp(0.75)).Children(func() {
								label := months[week.Format("2006-01-02")]
								if label != "" {
									month, _ := time.Parse("2006-01", label)
									ui.Text(c, usageText(fmt.Sprintf("%d月", month.Month()), month.Format("Jan"))).Height(17).FixedLineHeight(16).NoWrap().Tooltip(label).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
								} else {
									ui.Box(c).Height(17)
								}
								for n := 0; n < 7; n++ {
									date := week.AddDate(0, 0, n).Format("2006-01-02")
									day, ok := lookup[date]
									if !ok {
										ui.Box(c).Size(12, 12)
										continue
									}
									level := 0
									if day.Total > 0 {
										level = 1
										for _, q := range levels {
											if day.Total > q {
												level++
											}
										}
									}
									color := usageHeatColor(t, level)
									selectedDay := st.Range.Active() && date >= st.Range.Start && date <= st.Range.End
									ui.Column(c).Width(12).Children(func() {
										button := ui.Button(c.Key("day-"+date), "").Label(date+" UTC").Size(12, 12).Padding(0).BorderWidth(0).Radius(design.RadiusXS).Background(color).FocusRing(false).Cursor(ui.CursorPointer).Tooltip(dayTip(day, providers, settings))
										if st.Range.Active() && !selectedDay {
											button.Opacity(0.3)
										}
										if selectedDay && st.Range.Start == st.Range.End || button.Hovered() || button.Focused() {
											button.Opacity(1).DrawOver(func(p *ui.Painter, r ui.Rect) {
												p.Stroke(ui.Rect{X: r.X - 3, Y: r.Y - 3, W: r.W + 6, H: r.H + 6}, t.Ring, 5, 2)
											})
										}
										buttons[date] = button
										if button.Focused() {
											if *focusDay == "" && !button.Clicked() {
												focusTarget = days[len(days)-1].Date
											} else {
												*focusDay = date
											}
										}
										if button.Focused() || button.Hovered() {
											*detail = date
										}
										for _, key := range []ui.Key{ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown, ui.KeyHome, ui.KeyEnd} {
											if button.Shortcut(0, key) {
												focusTarget = calendarFocusTarget(date, key, days)
											}
										}
										if button.Shortcut(ui.Shift, ui.KeyEnter) || button.Shortcut(ui.Shift, ui.KeySpace) {
											m.Store.SelectStatsDay(date, true)
										}
										if button.Clicked() {
											*detail = date
											*focusDay = date
											m.Store.SelectStatsDay(date, button.ClickModifiers()&ui.Shift != 0)
										}
									})
								}
							})
						}
					})
				})
			})
			if button, ok := buttons[focusTarget]; ok {
				button.Focus()
				button.ScrollIntoView()
				*detail = focusTarget
				*focusDay = focusTarget
			}
			if day, ok := lookup[*detail]; ok {
				pagekit.Caption(c, dayTip(day, providers, settings)).FillWidth()
			}
			ui.Row(c).FillWidth().Gap(design.Sp(1)).AlignItems(ui.Center).Margin(design.Sp(3.5), 0, 0).Children(func() {
				ui.Spacer(c)
				pagekit.Caption(c, usageText("少", "Less"))
				for level := 0; level <= 4; level++ {
					ui.Box(c).Size(12, 12).Radius(design.RadiusXS).Background(usageHeatColor(t, level))
				}
				pagekit.Caption(c, usageText("多", "More"))
			})
		})
	})
}

// Weeks run across and Monday-first weekdays run down, matching the original heatmap. Missing dates stay inert.
func calendarFocusTarget(date string, key ui.Key, days []usagestore.DayUsage) string {
	if len(days) == 0 {
		return ""
	}
	if key == ui.KeyHome {
		return days[0].Date
	}
	if key == ui.KeyEnd {
		return days[len(days)-1].Date
	}
	delta := 0
	switch key {
	case ui.KeyLeft:
		delta = -7
	case ui.KeyRight:
		delta = 7
	case ui.KeyUp:
		delta = -1
	case ui.KeyDown:
		delta = 1
	default:
		return ""
	}
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	target := d.AddDate(0, 0, delta).Format("2006-01-02")
	for _, day := range days {
		if day.Date == target {
			return target
		}
	}
	return ""
}

type usageSlice struct {
	name    string
	value   float64
	color   ui.Color
	buckets usagestore.Totals
}

func modelSlices(t *design.Tokens, models []usagestore.ModelUsage) []usageSlice {
	palette := usageSeriesColors(t)
	rows := append([]usagestore.ModelUsage(nil), models...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total == rows[j].Total {
			return rows[i].Model < rows[j].Model
		}
		return rows[i].Total > rows[j].Total
	})
	slices := []usageSlice{}
	for i, row := range rows {
		if i < 5 {
			slices = append(slices, usageSlice{row.Model, row.Total, palette[i], row.Totals})
		} else if i == 5 {
			slices = append(slices, usageSlice{usageText("其他", "Other"), row.Total, palette[5], row.Totals})
		} else {
			slices[5].value += row.Total
			slices[5].buckets.Input += row.Input
			slices[5].buckets.Output += row.Output
			slices[5].buckets.CacheRead += row.CacheRead
			slices[5].buckets.CacheWrite += row.CacheWrite
		}
	}
	return slices
}

func stackedUsageView(c *ui.Context, days []usagestore.DayUsage, models []usagestore.ModelUsage, providers map[string]string, settings usagestore.BillingSettings) {
	t := design.For(c)
	slices := modelSlices(t, models)
	maxTotal := 0.0
	for _, day := range days {
		maxTotal = math.Max(maxTotal, day.Total)
	}
	ui.Row(c).FillWidth().Wrap().Gap(design.Sp(3)).AlignItems(ui.Center).Children(func() {
		for _, s := range slices {
			ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(8, 8).Radius(design.RadiusXS).Background(s.color).Shrink(0)
				pagekit.Caption(c, s.name)
			})
		}
	})
	usageTimeline(c, days, slices, maxTotal, providers, settings)
}

func modelShareView(c *ui.Context, models []usagestore.ModelUsage, total float64, settings usagestore.BillingSettings) {
	t := design.For(c)
	slices := modelSlices(t, models)
	tip := fmt.Sprintf("%.0f tokens", total)
	for _, s := range slices {
		percent := 0.0
		if total > 0 {
			percent = s.value / total * 100
		}
		tip += fmt.Sprintf("\n%s · %.1f%% · %.0f tokens · %s", s.name, percent, s.value, usageHitRate(s.buckets))
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Children(func() {
		ui.Box(c).Width(180).Height(180).Tooltip(tip).Draw(func(p *ui.Painter, r ui.Rect) {
			if total <= 0 {
				p.Fill(r, t.Muted, 80)
				return
			}
			angle := -math.Pi / 2
			for _, s := range slices {
				sweep := s.value / total * 2 * math.Pi
				if sweep <= 0 {
					continue
				}
				var path ui.Path
				steps := max(2, int(math.Ceil(sweep*30)))
				for i := 0; i <= steps; i++ {
					a := angle + sweep*float64(i)/float64(steps)
					x, y := r.X+r.W/2+float32(math.Cos(a))*82, r.Y+r.H/2+float32(math.Sin(a))*82
					if i == 0 {
						path.MoveTo(x, y)
					} else {
						path.LineTo(x, y)
					}
				}
				for i := steps; i >= 0; i-- {
					a := angle + sweep*float64(i)/float64(steps)
					path.LineTo(r.X+r.W/2+float32(math.Cos(a))*58, r.Y+r.H/2+float32(math.Sin(a))*58)
				}
				path.Close()
				p.FillPath(&path, s.color)
				angle += sweep
			}
			usagePlotText(p, fmt.Sprintf("%.0f", total), ui.Rect{X: r.X + 25, Y: r.Y + 68, W: 130, H: 25}, theme.InterfaceSize(18), t.Foreground, ui.Center)
			usagePlotText(p, usageText("总 Tokens", "Total tokens"), ui.Rect{X: r.X + 25, Y: r.Y + 94, W: 130, H: 18}, theme.InterfaceSize(11), t.MutedForeground, ui.Center)
		})
		ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Children(func() {
			for _, s := range slices {
				percent := 0.0
				if total > 0 {
					percent = s.value / total * 100
				}
				ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(8, 8).Radius(design.RadiusXS).Background(s.color).Shrink(0)
					pagekit.Caption(c, fmt.Sprintf("%s · %.1f%% · %.0f tokens · %s", s.name, percent, s.value, usageHitRate(s.buckets))).Grow(1).MinWidth(0)
				})
			}
		})
		coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
				for _, row := range models {
					percent := 0.0
					if total > 0 {
						percent = row.Total / total * 100
					}
					ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
						pagekit.Body(c, fmt.Sprintf("%s · %s · %.1f%% · %.0f tokens", row.Provider, row.Model, percent, row.Total)).FillWidth()
						detail := totalsText(row.Totals)
						if phases := phaseTotalsText(row.Cost); phases != "" {
							detail += "\n" + phases
						}
						pagekit.Caption(c, detail+"\n"+usageCostText([]usagestore.ModelUsage{row}, settings)).FillWidth()
					})
				}
			})
		}).FillWidth().Height(180)
	})
}

func kpiMemoVersion(m Model, o *usagestore.Overview, models []usagestore.ModelUsage, billing usagestore.BillingView, columns int) uint64 {
	v := usageMemoVersion(m).Uint(uint64(columns)).Uint(uint64(o.UpdatedAt)).Uint(uint64(o.AllTime.SessionCount)).
		Uint(uint64(o.Coverage.UsageSessionsMain)).Uint(uint64(o.Coverage.UsageSessionsSubagent)).
		Bool(billing.Loaded).Bool(billing.Settings.PeakValleyEnabled)
	tot := o.AllTime.Totals
	for _, f := range []float64{tot.Input, tot.Output, tot.CacheRead, tot.CacheWrite, tot.Total} {
		v = mixFloat64(v, f)
	}
	settings := effectiveBillingSettings(billing)
	for _, row := range models {
		v = v.String(row.Provider).String(row.Model)
		v = mixFloat64(v, row.Total)
		v = mixFloat64(v, row.Input)
		v = mixFloat64(v, row.Output)
		v = mixFloat64(v, row.CacheRead)
		v = mixFloat64(v, row.CacheWrite)
		if row.Cost != nil {
			v = v.Bool(true)
			if row.Cost.Peak != nil {
				v = mixFloat64(mixFloat64(mixFloat64(v, row.Cost.Peak.Input), row.Cost.Peak.Output), row.Cost.Peak.CacheRead)
			}
			if row.Cost.OffPeak != nil {
				v = mixFloat64(mixFloat64(mixFloat64(v, row.Cost.OffPeak.Input), row.Cost.OffPeak.Output), row.Cost.OffPeak.CacheRead)
			}
		} else {
			v = v.Bool(false)
		}
		if price, ok := usagestore.ResolveModelPrice(row.Provider, row.Model, settings.Prices); ok {
			v = v.Bool(true).Bool(price.Flat)
			v = mixFloat64(mixFloat64(mixFloat64(v, price.InputCacheHit), price.InputCacheMiss), price.Output)
			if price.Idle != nil {
				v = mixFloat64(mixFloat64(mixFloat64(v, price.Idle.InputCacheHit), price.Idle.InputCacheMiss), price.Idle.Output)
			}
		} else {
			v = v.Bool(false)
		}
	}
	return v.Key()
}

func usageKPIView(c *ui.Context, m Model, o *usagestore.Overview, models []usagestore.ModelUsage, billing usagestore.BillingView) {
	settings := effectiveBillingSettings(billing)
	tot := o.AllTime.Totals
	input := tot.Input + tot.CacheRead + tot.CacheWrite
	rate := usageText("无输入数据", "No input data")
	if input > 0 {
		rate = fmt.Sprintf("%.2f%%", tot.CacheRead/input*100)
	}
	topName, topDetail := "—", ""
	if len(models) > 0 {
		top := models[0]
		for _, row := range models {
			if row.Total > top.Total {
				top = row
			}
		}
		topName = top.Model
		if tot.Total > 0 {
			topDetail = fmt.Sprintf("%.1f%%", top.Total/tot.Total*100)
		}
	}
	costValue := "—"
	costDetail := usageText("暂无已定价模型", "No priced models yet")
	peakSum, idleSum, priced := 0.0, 0.0, 0
	for _, row := range models {
		p, i, ok := usagestore.EstimateModelPhases(row, settings)
		if ok {
			peakSum += p
			idleSum += i
			priced++
		}
	}
	if priced > 0 {
		costValue = fmt.Sprintf("≈ ¥%.2f", (peakSum+idleSum)/100)
		costDetail = fmt.Sprintf(usageText("峰段 ¥%.2f · 谷段 ¥%.2f", "Peak ¥%.2f · Off-peak ¥%.2f"), peakSum/100, idleSum/100)
	}
	metrics := []struct{ label, value, detail string }{
		{usageText("总 Tokens", "Total tokens"), fmt.Sprintf("%.0f", tot.Total), fmt.Sprintf(usageText("输入 %.0f · 输出 %.0f · 缓存读取 %.0f", "Input %.0f · Output %.0f · Cache read %.0f"), input, tot.Output, tot.CacheRead)},
		{usageText("估算费用", "Estimated cost"), costValue, costDetail},
		{usageText("会话", "Sessions"), fmt.Sprint(o.AllTime.SessionCount), fmt.Sprintf(usageText("主会话 %d · 子代理 %d", "Main %d · Subagent %d"), o.Coverage.UsageSessionsMain, o.Coverage.UsageSessionsSubagent)},
		{usageText("最常用模型", "Top model"), topName, topDetail},
		{usageText("缓存命中率", "Cache hit rate"), rate, fmt.Sprintf(usageText("缓存读取 %.0f · 缓存写入 %.0f", "Cache read %.0f · Cache write %.0f"), tot.CacheRead, tot.CacheWrite)},
	}
	grid := ui.Grid(c.Key("usage-kpis")).FillWidth().MinWidth(0).Gap(design.Sp(3))
	width := grid.Bounds().W
	columns := 6
	if width <= 320 {
		columns = 1
	} else if width <= 460 {
		columns = 2
	}
	version := kpiMemoVersion(m, o, models, billing, columns)
	grid.Columns(columns).Memo(version).Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	}).Children(func() {
		t := design.For(c)
		for index, metric := range metrics {
			span := 1
			if columns == 6 {
				span = 2
				if index < 2 {
					span = 3
				}
			}
			ui.Column(c).ColumnSpan(span).MinWidth(0).Gap(design.Sp(1)).Padding(design.Sp(3), design.Sp(4)).Radius(design.RadiusXL).Background(t.Card).Border(design.BorderWidth, t.Border).Children(func() {
				pagekit.Caption(c, metric.label).FillWidth()
				ts := design.TextBase
				if index == 1 || index >= 3 {
					ts = design.TextSM
				}
				ui.Text(c, metric.value).FillWidth().FontSize(ts.Size).FixedLineHeight(ts.LineHeight).FontWeight(design.WeightBold).FontFeatures("tnum").TextColor(t.Foreground)
				pagekit.Caption(c, metric.detail).FillWidth().FontFeatures("tnum")
			})
		}
	})
}
