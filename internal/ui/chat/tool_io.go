package chat

import (
	"math"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// toolIORows groups generic evidence into independently scrollable input/output
// sections. Specialized bodies keep their own clipboard affordances.
func toolIORows(c *ui.Context, info ToolCardInfo, st ToolCardStyle, key string) {
	if info.Arguments == "" && info.Output == "" {
		return
	}
	border := float32(max(1, math.Floor(.5*float64(c.DeviceScale()))) / float64(c.DeviceScale()))
	background, stroke, secondary, caption := st.Field, st.Border.Alpha(.4), st.Muted, st.Muted.Alpha(.6)
	if st.Colors.CodeBlock != "" {
		background = chrome.Color(st.Colors.CodeBlock)
		stroke = chrome.Color(st.Colors.BorderL1)
		secondary = chrome.Color(st.Colors.Secondary)
		caption = chrome.Color(st.Colors.Caption)
	}
	ui.Column(c.Key("tool-io:"+key)).MinWidth(0).Shrink(0).Margin(4, 0, 4, 4).Label(i18n.T(i18n.KeyToolRowIOGroup)).
		Border(border, stroke).Radius(16).Background(background).Children(func() {
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
				ui.Row(c).FillWidth().MinWidth(0).Padding(12, 16, 12, 16).Gap(14).AlignItems(ui.Start).Children(func() {
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
			ui.Box(c).FillWidth().Height(.5).Shrink(0).Background(st.Border)
		}
		if info.Output != "" {
			ink := secondary
			if toolRowStatus(info.Status) == statusError {
				ink = st.Err
			}
			section(i18n.T(i18n.KeyChatToolIOOutput), info.Output, "output", ink)
		}
	})
}
