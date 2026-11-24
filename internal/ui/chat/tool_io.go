package chat

import (
	"math"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// toolIORows groups generic evidence into independently scrollable input/output
// sections. Specialized bodies keep their own clipboard affordances.
func toolIORows(c *ui.Context, info ToolCardInfo, _ ToolCardStyle, key string) {
	if info.Arguments == "" && info.Output == "" {
		return
	}
	t := design.For(c)
	border := float32(max(1, math.Floor(.5*float64(c.DeviceScale()))) / float64(c.DeviceScale()))
	background, stroke, secondary, caption := t.Code, t.Border, t.Foreground, t.MutedForeground
	ui.Column(c.Key("tool-io:"+key)).MinWidth(0).Shrink(0).Margin(design.Sp(1), 0, design.Sp(1), design.Sp(1)).Label(i18n.T(i18n.KeyToolRowIOGroup)).
		Background(background).Radius(design.RadiusXL2).Border(border, stroke).Children(func() {
		section := func(label, value, suffix string, color ui.Color) {
			font := theme.RenderPreferences().Code.Family
			labelLayout := ui.ShapeText(label, ui.Font{Family: font, Size: 11}, 0)
			width, labelHeight := labelLayout.Size()
			var viewport, gutter ui.Rect
			scroll := ui.Scroll(c.Key(suffix)).FillWidth().MinWidth(0).MaxHeight(174).Label(label).
				ObserveLayout(func(r ui.Rect) { viewport = r })
			state := ui.Local(scroll, "position", func() ui.ScrollState { return ui.ScrollState{} })
			scroll.TrackScroll(state).DrawOver(func(p *ui.Painter, _ ui.Rect) {
				p.Clip(viewport, 0, func() {
					p.TextLayout(labelLayout, gutter.X, max(gutter.Y, viewport.Y)+(16-labelHeight)/2, caption)
				})
			})
			scroll.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Padding(design.Sp(3), design.Sp(4), design.Sp(3), design.Sp(4)).Gap(design.Sp(3.5)).AlignItems(ui.Start).Children(func() {
					// A separate fixed gutter supplies the label after scrolling;
					// the text itself remains selectable and wraps in its column.
					ui.Box(c).Width(width).Height(16).Shrink(0).ObserveLayout(func(r ui.Rect) { gutter = r })
					ui.Text(c, value).Font(font).FontSize(11).FixedLineHeight(16).TextColor(color).Grow(1).MinWidth(0).Selectable()
				})
			})
		}
		if info.Arguments != "" {
			section(i18n.T(i18n.KeyChatToolInput), prettyToolPayload(info.Arguments), "input", secondary)
		}
		if info.Arguments != "" && info.Output != "" {
			coss.Separator(c, coss.Horizontal)
		}
		if info.Output != "" {
			ink := secondary
			if toolRowStatus(info.Status) == statusError {
				ink = t.DestructiveForeground
			}
			section(i18n.T(i18n.KeyChatToolIOOutput), info.Output, "output", ink)
		}
	})
}
