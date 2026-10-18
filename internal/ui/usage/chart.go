package usage

import (
	"fmt"
	"math"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// Bar is one column in a simple usage chart.
type Bar struct {
	Label string
	Value float64
}

// BarChartView paints a compact vertical bar chart (native M9 stand-in for
// dsh-usage-panel heatmaps until full series RPCs exist).
func BarChartView(c *ui.Context, bars []Bar, fg, muted, accent, field ui.Color) {
	if len(bars) == 0 {
		return
	}
	max := 0.0
	for _, b := range bars {
		if b.Value > max {
			max = b.Value
		}
	}
	if max <= 0 {
		max = 1
	}
	const chartH = float32(96)
	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.End).Height(chartH + 28).Children(func() {
			for _, b := range bars {
				h := float32(math.Round(float64(chartH) * (b.Value / max)))
				if b.Value > 0 && h < 4 {
					h = 4
				}
				ui.Column(c).Gap(4).AlignItems(ui.Center).Width(36).Children(func() {
					ui.Box(c).Height(chartH - h).Width(22)
					ui.Box(c).Height(h).Width(22).Background(accent).Radius(3)
					ui.Text(c, b.Label).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
					ui.Text(c, fmt.Sprintf("%.0f", b.Value)).FontSize(fonttheme.InterfaceSize(10)).TextColor(fg)
				})
			}
		})
		ui.Box(c).Fill().Height(1).Background(field)
	})
}
