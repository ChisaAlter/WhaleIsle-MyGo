package schedule

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

func scheduleValueFace(c *ui.Context, value string, active bool, kind ...string) {
	t := design.For(c)
	face := ui.Row(c).MinWidth(0).MinHeight(design.Sp(8)).Padding(0, design.Sp(2)).Gap(design.Sp(1.5)).Radius(design.RadiusFull).AlignItems(ui.Center)
	if active {
		face.Background(t.Accent)
	}
	face.Children(func() {
		pagekit.Body(c, value).MinWidth(0)
		ic := coss.IconChevronDown
		if len(kind) > 0 {
			if kind[0] == "date" {
				ic = coss.IconCalendar
			}
			if kind[0] == "clock" {
				ic = coss.IconClock
			}
		}
		ui.Icon(c, ic.SVG()).Size(design.IconXS, design.IconXS).TextColor(t.MutedForeground)
	})
}

func scheduleRuleRow(c *ui.Context, label string, last bool, body func()) {
	t := design.For(c)
	row := ui.Row(c).FillWidth().MinWidth(0).MinHeight(design.Sp(12)).Gap(design.Sp(3)).AlignItems(ui.Center)
	if !last {
		row.BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border)
	}
	row.Children(func() {
		pagekit.Body(c, label).Shrink(0)
		ui.Spacer(c)
		body()
	})
}
