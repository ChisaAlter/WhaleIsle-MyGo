package schedule

import "github.com/egoist/mygo/ui"

var scheduleChevron = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
var scheduleCalendar = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><rect x="2" y="3" width="12" height="11" rx="2" stroke="currentColor"/><path d="M5 1.5v3M11 1.5v3M2 7h12" stroke="currentColor"/></svg>`))

func scheduleValueFace(c *ui.Context, value string, active bool, kind ...string) {
	face := ui.Row(c).MinWidth(0).MinHeight(32).Padding(0, 8).Gap(6).Radius(18).AlignItems(ui.Center)
	if active {
		face.Background(c.Theme().SurfaceHover)
	}
	face.Children(func() {
		ui.Text(c, value).MinWidth(0).FontSize(14)
		icon := scheduleChevron
		if len(kind) > 0 {
			if kind[0] == "date" {
				icon = scheduleCalendar
			}
			if kind[0] == "clock" {
				icon = scheduleIcons["clock"]
			}
		}
		ui.Icon(c, icon).Size(14, 14).TextColor(c.Theme().TextMuted)
	})
}

func scheduleRuleRow(c *ui.Context, label string, last bool, body func()) {
	row := ui.Row(c).FillWidth().MinWidth(0).MinHeight(48).Gap(12).AlignItems(ui.Center).Border(.5, c.Theme().Border).BorderWidth(0, 0, .5, 0)
	if last {
		row.BorderWidth(0)
	}
	row.Children(func() { ui.Text(c, label).FontSize(14).Shrink(0); ui.Box(c).Grow(1); body() })
}
