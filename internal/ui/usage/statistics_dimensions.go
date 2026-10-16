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
	return fmt.Sprintf("≈ CNY %.2f (peak %.2f / off-peak %.2f) · %d priced / %d unpriced", (peak+idle)/100, peak/100, idle/100, priced, unknown)
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
		lines = append(lines, fmt.Sprintf("%s / %s · %.0f\n%s\n%s", row.Provider, row.Model, row.Total, totalsText(row.Totals), usageCostText([]usagestore.ModelUsage{row}, settings)))
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
	usageKPIView(c, o.AllTime.Totals, o.AllTime.ByModel, settings, fg, muted)
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
	ui.Text(c, title).Bold().TextColor(fg)
	stackedUsageView(c, days, models, providers, settings, fg, muted, field)
	shareModels := o.AllTime.ByModel
	shareTotal := o.AllTime.Totals.Total
	title = usageText("模型占比（全部历史）", "Model share (all time)")
	if st.Range.Active() {
		selected := st.Selection()
		shareModels = selected.Models
		shareTotal = selected.Totals.Total
		title = usageText("模型占比（选定日期）", "Model share (selected dates)")
	}
	ui.Text(c, title).Bold().TextColor(fg)
	modelShareView(c, shareModels, shareTotal, settings, fg, muted, field)
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
	ui.Text(c, usageText("用量日历（UTC）", "Usage calendar (UTC)")).Bold().TextColor(fg)
	ui.Text(c, fmt.Sprintf("%.0f tokens · %d active / %d days · %s", selected.Totals.Total, active, len(selected.Days), usageCostText(selected.Models, settings))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	ui.Text(c, usageText("颜色按全窗口四分位分级；悬停查看模型、分项 Token 与峰谷费用。点击选一天，已有选区时可扩展。", "Colors use whole-window quartiles. Hover for models, token buckets and peak/off-peak costs. Pick a day or extend the current range.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	start, _ := time.Parse("2006-01-02", days[0].Date)
	end, _ := time.Parse("2006-01-02", days[len(days)-1].Date)
	start = start.AddDate(0, 0, -(int(start.Weekday())+6)%7)
	ui.Scroll(c).Height(220).Children(func() {
		ui.Column(c).Gap(5).Children(func() {
			ui.Text(c, "Mon       Tue       Wed       Thu       Fri       Sat       Sun").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			month := ""
			for week := start; !week.After(end); week = week.AddDate(0, 0, 7) {
				label := week.Format("2006-01")
				if label != month {
					month = label
					ui.Text(c, label).TextColor(fg)
				}
				ui.Row(c).Gap(3).Children(func() {
					for n := 0; n < 7; n++ {
						date := week.AddDate(0, 0, n).Format("2006-01-02")
						day, ok := lookup[date]
						if !ok {
							ui.Box(c).Width(48).Height(30)
							continue
						}
						color := field
						if day.Total > 0 {
							level := 1
							for _, q := range levels {
								if day.Total > q {
									level++
								}
							}
							color = accent
							color.A = uint8(75 + level*45)
						}
						text := date[8:]
						if st.Range.Active() && date >= st.Range.Start && date <= st.Range.End {
							text = "✓" + text
						}
						ui.Column(c).Width(48).Children(func() {
							if ui.Button(c, text).Width(48).Background(color).Tooltip(dayTip(day, providers, settings)).Clicked() {
								m.Store.SelectStatsDay(date, false)
							}
							if st.Range.Active() && ui.Button(c, "+").Width(48).Tooltip(usageText("扩展选区", "Extend selection")).Clicked() {
								m.Store.SelectStatsDay(date, true)
							}
						})
					}
				})
			}
		})
	})
}

type usageSlice struct {
	name  string
	value float64
	color ui.Color
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
			slices = append(slices, usageSlice{row.Model, row.Total, usageSeriesColors[i]})
		} else if i == 5 {
			slices = append(slices, usageSlice{usageText("其他", "Other"), row.Total, usageSeriesColors[5]})
		} else {
			slices[5].value += row.Total
		}
	}
	return slices
}

func stackedUsageView(c *ui.Context, days []usagestore.DayUsage, models []usagestore.ModelUsage, providers map[string]string, settings usagestore.BillingSettings, fg, muted, field ui.Color) {
	slices := modelSlices(models)
	maxTotal := 1.0
	for _, day := range days {
		maxTotal = math.Max(maxTotal, day.Total)
	}
	for _, s := range slices {
		ui.Text(c, "■ "+s.name).FontSize(fonttheme.InterfaceSize(11)).TextColor(s.color)
	}
	ui.Scroll(c).Height(200).Children(func() {
		ui.Column(c).Gap(8).Children(func() {
			for start := 0; start < len(days); start += 7 {
				ui.Row(c).Gap(5).AlignItems(ui.End).Children(func() {
					for _, day := range days[start:min(start+7, len(days))] {
						day := day
						ui.Column(c).Width(40).Gap(3).Children(func() {
							ui.Box(c).Width(36).Height(100).Tooltip(dayTip(day, providers, settings)).Draw(func(p *ui.Painter, r ui.Rect) {
								y := r.Y + r.H
								for i, s := range slices {
									value := day.Models[s.name].Total
									if i == 5 {
										value = 0
										for name, t := range day.Models {
											known := false
											for _, top := range slices[:5] {
												if top.name == name {
													known = true
													break
												}
											}
											if !known {
												value += t.Total
											}
										}
									}
									height := float32(value/maxTotal) * r.H
									y -= height
									p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: height}, s.color, 0)
								}
								if day.Total == 0 {
									p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 2, W: r.W, H: 2}, field, 0)
								}
							})
							ui.Text(c, day.Date[5:]).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
							ui.Text(c, fmt.Sprintf("%.0f", day.Total)).FontSize(fonttheme.InterfaceSize(10)).TextColor(fg)
						})
					}
				})
			}
		})
	})
}

func modelShareView(c *ui.Context, models []usagestore.ModelUsage, total float64, settings usagestore.BillingSettings, fg, muted, field ui.Color) {
	slices := modelSlices(models)
	tip := fmt.Sprintf("%.0f tokens", total)
	for _, s := range slices {
		percent := 0.0
		if total > 0 {
			percent = s.value / total * 100
		}
		tip += fmt.Sprintf("\n%s · %.1f%% · %.0f tokens", s.name, percent, s.value)
	}
	ui.Box(c).Width(160).Height(160).Tooltip(tip).Draw(func(p *ui.Painter, r ui.Rect) {
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
				x, y := r.X+r.W/2+float32(math.Cos(a))*r.W/2, r.Y+r.H/2+float32(math.Sin(a))*r.H/2
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			for i := steps; i >= 0; i-- {
				a := angle + sweep*float64(i)/float64(steps)
				path.LineTo(r.X+r.W/2+float32(math.Cos(a))*r.W*.30, r.Y+r.H/2+float32(math.Sin(a))*r.H*.30)
			}
			path.Close()
			p.FillPath(&path, s.color)
			angle += sweep
		}
	})
	for _, s := range slices {
		percent := 0.0
		if total > 0 {
			percent = s.value / total * 100
		}
		ui.Text(c, fmt.Sprintf("■ %s · %.1f%% · %.0f tokens", s.name, percent, s.value)).FontSize(fonttheme.InterfaceSize(11)).TextColor(s.color)
	}
	ui.Scroll(c).Height(180).Children(func() {
		ui.Column(c).Gap(6).Children(func() {
			for _, row := range models {
				percent := 0.0
				if total > 0 {
					percent = row.Total / total * 100
				}
				ui.Text(c, fmt.Sprintf("%s / %s · %.1f%% · %.0f tokens", row.Provider, row.Model, percent, row.Total)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				ui.Text(c, totalsText(row.Totals)+"\n"+usageCostText([]usagestore.ModelUsage{row}, settings)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
	})
}

func usageKPIView(c *ui.Context, t usagestore.Totals, models []usagestore.ModelUsage, settings usagestore.BillingSettings, fg, muted ui.Color) {
	input := t.Input + t.CacheRead + t.CacheWrite
	rate := usageText("无输入数据", "No input data")
	if input > 0 {
		rate = fmt.Sprintf("%.2f%%", t.CacheRead/input*100)
	}
	ui.Text(c, usageText("全部历史指标", "All-time indicators")).Bold().TextColor(fg)
	ui.Text(c, fmt.Sprintf("Billed input %.0f · Output %.0f · Cache hit %s", input, t.Output, rate)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	ui.Text(c, usageCostText(models, settings)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if len(models) > 0 {
		top := models[0]
		for _, row := range models {
			if row.Total > top.Total {
				top = row
			}
		}
		percent := 0.0
		if t.Total > 0 {
			percent = top.Total / t.Total * 100
		}
		ui.Text(c, fmt.Sprintf("Top model: %s / %s · %.1f%%", top.Provider, top.Model, percent)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
}

func providerUsageBar(c *ui.Context, value, maximum float64, accent, field ui.Color) {
	ui.Box(c).Width(280).Height(8).Background(field).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * float32(value/maximum), H: r.H}, accent, 3)
	})
}
