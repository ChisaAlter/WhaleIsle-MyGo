package usage

import (
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"
)

// HeatCell is one day cell in the activity heatmap.
type HeatCell struct {
	Label string
	Value float64
}

// HeatmapView paints a GitHub-style intensity row (native stand-in for
// dsh-usage-panel heatmaps).
func HeatmapView(c *ui.Context, cells []HeatCell, muted, accent, field ui.Color) {
	if len(cells) == 0 {
		return
	}
	max := 0.0
	for _, cell := range cells {
		if cell.Value > max {
			max = cell.Value
		}
	}
	ui.Row(c).Gap(4).AlignItems(ui.End).Children(func() {
		for _, cell := range cells {
			level := 0.0
			if max > 0 {
				level = cell.Value / max
			}
			fill := field
			switch {
			case cell.Value <= 0:
				fill = field
			case level < 0.25:
				fill = field.Mix(accent, 0.35)
			case level < 0.5:
				fill = field.Mix(accent, 0.55)
			case level < 0.75:
				fill = field.Mix(accent, 0.75)
			default:
				fill = accent
			}
			ui.Column(c).Gap(2).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(18, 18).Background(fill).Radius(3)
				ui.Text(c, cell.Label).FontSize(fonttheme.InterfaceSize(9)).TextColor(muted)
			})
		}
	})
}
