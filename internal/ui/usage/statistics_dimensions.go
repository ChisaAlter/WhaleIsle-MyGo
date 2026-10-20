package usage

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	usagestore "whaleisle/internal/store/usage"
)

var usageSeriesColors = []ui.Color{ui.Hex("#5B8FF9"), ui.Hex("#61DDAA"), ui.Hex("#F6BD16"), ui.Hex("#E8684A"), ui.Hex("#9270CA"), ui.Hex("#6F7B8B")}

func totalsText(t usagestore.Totals) string {
	return fmt.Sprintf("Input %.0f · Output %.0f · Cache read %.0f · Cache write %.0f", t.Input, t.Output, t.CacheRead, t.CacheWrite)
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
		lines = append(lines, fmt.Sprintf("%s · %s · %.0f\n%s\n%s", row.Provider, row.Model, row.Total, totalsText(row.Totals), usageCostText([]usagestore.ModelUsage{row}, settings)))
	}
	return strings.Join(lines, "\n")
}

func statisticsDimensionsView(c *ui.Context, m Model, st usagestore.StatsView, fg, muted, accent, field ui.Color) {
	o := st.Overview
	billing := m.Store.Billing()
	settings := billing.Settings
	if !billing.Loaded {
		settings = usagestore.BillingSettings{PeakValleyEnabled: true}
	}
	providers := map[string]string{}
	for _, row := range o.AllTime.ByModel {
		providers[row.Model] = row.Provider
	}
	usageKPIView(c, o, o.AllTime.ByModel, settings, fg, muted)
	calendarUsageView(c, m, st, providers, settings, fg, muted, accent, field)
	days := o.Days
	models := o.ByModel
	title := usageText("每日模型堆叠（近期）", "Daily model breakdown (recent)")
	if st.Range.Active() {
		selected := st.Selection()
		days = selected.Days
		models = selected.Models
		title = usageText("每日模型堆叠（选定日期）", "Daily model breakdown (selected dates)")
	} else if len(days) > st.Days {
		days = days[len(days)-st.Days:]
	}
	usageCard(c, "usage-daily-models", title, fg, func() {
		stackedUsageView(c, days, models, providers, settings, fg, muted, field)
	})
	shareModels := o.AllTime.ByModel
	shareTotal := o.AllTime.Totals.Total
	title = usageText("模型占比（全部历史）", "Model share (all time)")
	if st.Range.Active() {
		selected := st.Selection()
		shareModels = selected.Models
		shareTotal = selected.Totals.Total
		title = usageText("模型占比（选定日期）", "Model share (selected dates)")
	}
	usageCard(c, "usage-model-share", title, fg, func() {
		modelShareView(c, shareModels, shareTotal, settings, fg, muted, field)
	})
	if st.Range.Active() || len(o.Providers) > 0 && !(len(o.Providers) == 1 && (o.Providers[0].ID == "unknown" || o.Providers[0].Totals.Total <= 0)) {
		ui.Text(c, usageText("服务商分项用量", "Provider token breakdown")).Bold().TextColor(fg)
		providerMax := 1.0
		if st.Range.Active() {
			totals := map[string]float64{}
			for _, row := range shareModels {
				totals[row.Provider] += row.Total
			}
			for _, value := range totals {
				providerMax = math.Max(providerMax, value)
			}
		} else {
			for _, provider := range o.Providers {
				providerMax = math.Max(providerMax, provider.Totals.Total)
			}
		}
		if st.Range.Active() {
			groups := map[string][]usagestore.ModelUsage{}
			for _, row := range shareModels {
				groups[row.Provider] = append(groups[row.Provider], row)
			}
			keys := make([]string, 0, len(groups))
			for key := range groups {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				var t usagestore.Totals
				for _, row := range groups[key] {
					t.Input += row.Input
					t.Output += row.Output
					t.CacheRead += row.CacheRead
					t.CacheWrite += row.CacheWrite
					t.Total += row.Total
				}
				ui.Text(c, fmt.Sprintf("%s · %.0f tokens", key, t.Total)).TextColor(fg)
				providerUsageBar(c, t.Total, providerMax, accent, field)
				ui.Text(c, totalsText(t)+"\n"+usageCostText(groups[key], settings)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		} else {
			for _, provider := range o.Providers {
				name := provider.Name
				if name == "" {
					name = provider.ID
				}
				rows := []usagestore.ModelUsage{}
				for _, row := range shareModels {
					if row.Provider == provider.ID {
						rows = append(rows, row)
					}
				}
				ui.Text(c, fmt.Sprintf("%s · %.0f tokens", name, provider.Totals.Total)).TextColor(fg)
				providerUsageBar(c, provider.Totals.Total, providerMax, accent, field)
				ui.Text(c, totalsText(provider.Totals)+"\n"+usageCostText(rows, settings)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		}
	}
	ui.Text(c, usageText("每日明细（与上方趋势范围相同）", "Daily details (same range as trend)")).Bold().TextColor(fg)
	ui.Scroll(c).Height(220).Children(func() {
		ui.Column(c).Gap(8).Children(func() {
			for _, day := range days {
				ui.Text(c, dayTip(day, providers, settings)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
	})
}

// Calendar cells use one set of quartiles over all returned days, including
// when a date selection is active. Gaps are blank rather than zero-use days.
func calendarUsageView(c *ui.Context, m Model, st usagestore.StatsView, providers map[string]string, settings usagestore.BillingSettings, fg, muted, accent, field ui.Color) {
	days := append([]usagestore.DayUsage(nil), st.Overview.Days...)
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	if len(days) == 0 {
		return
	}
	usageCard(c, "usage-calendar-card", "", fg, func() {
		surface, _, secondary := usageCardPalette(c)
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
		heading := ui.Text(c.Key("usage-calendar"), usageText("用量日历（UTC）", "Usage calendar (UTC)")).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(600).TextColor(fg)
		detail := ui.Local(heading, "detail-date", func() string { return "" })
		focusDay := ui.Local(heading, "focus-date", func() string { return "" })
		buttons := map[string]ui.Element{}
		focusTarget := ""
		ui.Text(c, fmt.Sprintf(usageText("%.0f tokens · %d 天有用量，共 %d 天 · %s", "%.0f tokens · %d active of %d days · %s"), selected.Totals.Total, active, len(selected.Days), usageCostText(selected.Models, settings))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, usageText("颜色按全窗口四分位分级；聚焦或悬停查看日期详情；方向键按日/周移动，Home/End 到首尾；Enter 选择，Shift+Enter 扩展。", "Colors use whole-window quartiles. Focus or hover for day details. Arrow keys move by day/week; Home/End reach bounds. Enter selects; Shift+Enter extends.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		start, _ := time.Parse("2006-01-02", days[0].Date)
		end, _ := time.Parse("2006-01-02", days[len(days)-1].Date)
		start = start.AddDate(0, 0, -(int(start.Weekday())+6)%7)
		months := calendarMonthLabels(start, end, lookup)
		ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Height(124).FocusGroup(ui.Horizontal | ui.Vertical).Children(func() {
			ui.Row(c).Gap(6).Children(func() {
				ui.Column(c).Width(20).Gap(3).Children(func() {
					ui.Box(c).Height(17)
					for _, label := range []string{usageText("一", "Mon"), "", usageText("三", "Wed"), "", usageText("五", "Fri"), "", ""} {
						ui.Text(c, label).Height(12).FixedLineHeight(12).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted).NoWrap()
					}
				})
				ui.Row(c).Gap(3).Children(func() {
					for week := start; !week.After(end); week = week.AddDate(0, 0, 7) {
						ui.Column(c).Width(12).Gap(3).Children(func() {
							label := months[week.Format("2006-01-02")]
							if label != "" {
								month, _ := time.Parse("2006-01", label)
								ui.Text(c, usageText(fmt.Sprintf("%d月", month.Month()), month.Format("Jan"))).Height(17).FixedLineHeight(16).NoWrap().Tooltip(label).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
								color := usageHeatColor(level, surface, secondary)
								selectedDay := st.Range.Active() && date >= st.Range.Start && date <= st.Range.End
								ui.Column(c).Width(12).Children(func() {
									button := ui.Button(c.Key("day-"+date), "").Label(date+" UTC").Size(12, 12).Padding(0).BorderWidth(0).Radius(3).Background(color).FocusRing(false).Tooltip(dayTip(day, providers, settings))
									if st.Range.Active() && !selectedDay {
										button.Opacity(.3)
									}
									if selectedDay && st.Range.Start == st.Range.End || button.Hovered() || button.Focused() {
										button.Opacity(1).DrawOver(func(p *ui.Painter, r ui.Rect) {
											p.Stroke(ui.Rect{X: r.X - 3, Y: r.Y - 3, W: r.W + 6, H: r.H + 6}, accent, 5, 2)
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
			ui.Text(c, dayTip(day, providers, settings)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Margin(14, 0, 0).Children(func() {
			ui.Spacer(c)
			ui.Text(c, usageText("少", "Less")).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
			for level := 0; level <= 4; level++ {
				ui.Box(c).Size(12, 12).Radius(3).Background(usageHeatColor(level, surface, secondary))
			}
			ui.Text(c, usageText("多", "More")).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
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

func modelSlices(models []usagestore.ModelUsage) []usageSlice {
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
			slices = append(slices, usageSlice{row.Model, row.Total, usageSeriesColors[i], row.Totals})
		} else if i == 5 {
			slices = append(slices, usageSlice{usageText("其他", "Other"), row.Total, usageSeriesColors[5], row.Totals})
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

func stackedUsageView(c *ui.Context, days []usagestore.DayUsage, models []usagestore.ModelUsage, providers map[string]string, settings usagestore.BillingSettings, fg, muted, field ui.Color) {
	slices := modelSlices(models)
	maxTotal := 0.0
	for _, day := range days {
		maxTotal = math.Max(maxTotal, day.Total)
	}
	ui.Row(c).FillWidth().Wrap().Gap(12).Children(func() {
		for _, s := range slices {
			ui.Text(c, "■ "+s.name).FontSize(fonttheme.InterfaceSize(11)).TextColor(s.color)
		}
	})
	usageTimeline(c, days, slices, maxTotal, providers, settings, muted, field)
}

func modelShareView(c *ui.Context, models []usagestore.ModelUsage, total float64, settings usagestore.BillingSettings, fg, muted, field ui.Color) {
	slices := modelSlices(models)
	tip := fmt.Sprintf("%.0f tokens", total)
	for _, s := range slices {
		percent := 0.0
		if total > 0 {
			percent = s.value / total * 100
		}
		tip += fmt.Sprintf("\n%s · %.1f%% · %.0f tokens · %s", s.name, percent, s.value, usageHitRate(s.buckets))
	}
	ui.Box(c).Width(180).Height(180).Tooltip(tip).Draw(func(p *ui.Painter, r ui.Rect) {
		if total <= 0 {
			p.Fill(r, field, 80)
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
		usagePlotText(p, fmt.Sprintf("%.0f", total), ui.Rect{X: r.X + 25, Y: r.Y + 68, W: 130, H: 25}, fonttheme.InterfaceSize(18), fg, ui.Center)
		usagePlotText(p, usageText("总 Tokens", "Total tokens"), ui.Rect{X: r.X + 25, Y: r.Y + 94, W: 130, H: 18}, fonttheme.InterfaceSize(11), muted, ui.Center)
	})
	for _, s := range slices {
		percent := 0.0
		if total > 0 {
			percent = s.value / total * 100
		}
		ui.Text(c, fmt.Sprintf("■ %s · %.1f%% · %.0f tokens · %s", s.name, percent, s.value, usageHitRate(s.buckets))).FontSize(fonttheme.InterfaceSize(11)).TextColor(s.color)
	}
	ui.Scroll(c).Height(180).Children(func() {
		ui.Column(c).Gap(6).Children(func() {
			for _, row := range models {
				percent := 0.0
				if total > 0 {
					percent = row.Total / total * 100
				}
				ui.Text(c, fmt.Sprintf("%s · %s · %.1f%% · %.0f tokens", row.Provider, row.Model, percent, row.Total)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				ui.Text(c, totalsText(row.Totals)+"\n"+usageCostText([]usagestore.ModelUsage{row}, settings)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
	})
}

func usageKPIView(c *ui.Context, o *usagestore.Overview, models []usagestore.ModelUsage, settings usagestore.BillingSettings, fg, muted ui.Color) {
	t := o.AllTime.Totals
	input := t.Input + t.CacheRead + t.CacheWrite
	rate := usageText("无输入数据", "No input data")
	if input > 0 {
		rate = fmt.Sprintf("%.2f%%", t.CacheRead/input*100)
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
		if t.Total > 0 {
			topDetail = fmt.Sprintf("%.1f%%", top.Total/t.Total*100)
		}
	}
	costValue := "—"
	sum, priced := 0.0, 0
	for _, row := range models {
		p, i, ok := usagestore.EstimateModelPhases(row, settings)
		if ok {
			sum += p + i
			priced++
		}
	}
	if priced > 0 {
		costValue = fmt.Sprintf("≈ ¥%.2f", sum/100)
	}
	metrics := []struct{ label, value, detail string }{
		{usageText("总 Tokens", "Total tokens"), fmt.Sprintf("%.0f", t.Total), fmt.Sprintf(usageText("输入 %.0f · 输出 %.0f", "Input %.0f · Output %.0f"), input, t.Output)},
		{usageText("估算费用", "Estimated cost"), costValue, usageText("按模型价格估算", "Estimated from model prices")},
		{usageText("会话", "Sessions"), fmt.Sprint(o.AllTime.SessionCount), fmt.Sprintf(usageText("主会话 %d · 子代理 %d", "Main %d · Subagent %d"), o.Coverage.UsageSessionsMain, o.Coverage.UsageSessionsSubagent)},
		{usageText("最常用模型", "Top model"), topName, topDetail},
		{usageText("缓存命中率", "Cache hit rate"), rate, fmt.Sprintf(usageText("缓存读取 %.0f", "Cache read %.0f"), t.CacheRead)},
	}
	surface, border, secondary := usageCardPalette(c)
	grid := ui.Grid(c.Key("usage-kpis")).FillWidth().MinWidth(0).Gap(12)
	width := grid.Bounds().W
	columns := 6
	if width <= 320 {
		columns = 1
	} else if width <= 460 {
		columns = 2
	}
	grid.Columns(columns).Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
	}).Children(func() {
		for index, metric := range metrics {
			span := 1
			if columns == 6 {
				span = 2
				if index < 2 {
					span = 3
				}
			}
			ui.Column(c).ColumnSpan(span).MinWidth(0).Gap(0).Padding(12, 16).Radius(8).Background(surface).Border(1, border).Children(func() {
				ui.Text(c, metric.label).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(secondary)
				size, height := float32(16), float32(24)
				if index == 1 || index >= 3 {
					size, height = 14, 22
				}
				ui.Text(c, metric.value).FillWidth().Margin(8, 0, 0).FontSize(fonttheme.InterfaceSize(size)).FixedLineHeight(fonttheme.InterfaceSize(height)).FontWeight(700).FontFeatures("tnum").TextColor(fg)
				ui.Text(c, metric.detail).FillWidth().Margin(4, 0, 0).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).FontFeatures("tnum").TextColor(secondary)
			})
		}
	})
}
func providerUsageBar(c *ui.Context, value, maximum float64, accent, field ui.Color) {
	ui.Box(c).FillWidth().Height(8).Background(field).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * max(.02, float32(value/maximum)), H: r.H}, accent, 3)
	})
}
