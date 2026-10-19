package usage

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
	usagestore "whaleisle/internal/store/usage"
	fonttheme "whaleisle/internal/ui/theme"
)

func usageHitRate(t usagestore.Totals) string {
	input := t.Input + t.CacheRead + t.CacheWrite
	if input <= 0 {
		return usageText("缓存命中率 —", "Cache hit —")
	}
	return fmt.Sprintf("%s %.1f%%", usageText("缓存命中率", "Cache hit"), 100*t.CacheRead/input)
}

func usageNiceCeil(v float64) float64 {
	if v <= 0 {
		return 1
	}
	p := math.Pow(10, math.Floor(math.Log10(v)))
	d := v / p
	m := 10.0
	if d <= 1 {
		m = 1
	} else if d <= 2 {
		m = 2
	} else if d <= 5 {
		m = 5
	}
	return m * p
}

// One responsive plot retains the original 7/14/30-day comparison on a single
// time axis. Hit boxes match every date band, including zero-use dates.
func usageTimeline(c *ui.Context, days []usagestore.DayUsage, slices []usageSlice, maximum float64, providers map[string]string, settings usagestore.BillingSettings, muted, field ui.Color) {
	if len(days) == 0 {
		return
	}
	maximum = usageNiceCeil(maximum)
	chart := ui.Box(c).FillWidth().MinWidth(0).Height(230)
	width := chart.Bounds().W
	chart.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != width {
			c.Invalidate()
		}
		plot := ui.Rect{X: r.X + 52, Y: r.Y + 10, W: max(1, r.W-64), H: r.H - 36}
		for n := 0; n <= 4; n++ {
			y := plot.Y + plot.H - float32(n)*plot.H/4
			p.Fill(ui.Rect{X: plot.X, Y: y, W: plot.W, H: 1}, field, 0)
			usagePlotText(p, fmt.Sprintf("%.0f", maximum*float64(n)/4), ui.Rect{X: r.X, Y: y - 8, W: 46, H: 16}, fonttheme.InterfaceSize(10), muted, ui.End)
		}
		band := plot.W / float32(len(days))
		bar := min(float32(44), band*.6)
		for index, day := range days {
			x := plot.X + (float32(index)+.5)*band
			y := plot.Y + plot.H
			for i, s := range slices {
				value := day.Models[s.name].Total
				if i == 5 {
					value = 0
					for name, buckets := range day.Models {
						known := false
						for _, top := range slices[:5] {
							if top.name == name {
								known = true
								break
							}
						}
						if !known {
							value += buckets.Total
						}
					}
				}
				height := float32(value/maximum) * plot.H
				y -= height
				p.Fill(ui.Rect{X: x - bar/2, Y: y, W: bar, H: height}, s.color, 0)
			}
			step := max(1, int(math.Ceil(float64(len(days))/7)))
			if index%step == 0 || index == len(days)-1 {
				label := day.Date
				if len(label) > 5 {
					label = label[5:]
				}
				usagePlotText(p, label, ui.Rect{X: x - 22, Y: plot.Y + plot.H + 7, W: 44, H: 18}, fonttheme.InterfaceSize(10), muted, ui.Center)
			}
		}
	}).Children(func() {
		ui.Row(c).Absolute().Left(52).Right(12).Top(10).Bottom(26).Children(func() {
			for _, day := range days {
				ui.Box(c).Grow(1).Height(194).Tooltip(dayTip(day, providers, settings))
			}
		})
	})
}

func usagePlotText(p *ui.Painter, text string, r ui.Rect, size float32, color ui.Color, align ui.Align) {
	layout := ui.ShapeText(text, ui.Font{Family: fonttheme.TextFamily(fonttheme.RenderPreferences().Interface.Family), Size: size}, 0)
	w, h := layout.Size()
	x := r.X
	if align == ui.Center {
		x += (r.W - w) / 2
	} else if align == ui.End {
		x += r.W - w
	}
	p.TextLayout(layout, x, r.Y+(r.H-h)/2, color)
}
