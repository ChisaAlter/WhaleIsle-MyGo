package frame

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type headerNoticeAction struct {
	label    string
	disabled bool
	run      func()
}

// Source-owned notices leave the job/catalog reading surface in place. They
// disappear with that consumer instead of retaining callbacks into another Host.
func headerNotice(c *ui.Context, key, message string, colors theme.Colors, bottom float32, actions []headerNoticeAction, dismiss func()) {
	width, height := viewportDIP(c)
	size := max(float32(0), min(float32(520), width-32))
	// Keep the lifted notice attached to its real consumer. Without this seat,
	// a notice over a jobs popover is mistaken for an outside press: pointer
	// down closes its owner before Retry/Confirm can receive pointer up.
	anchor := ui.Box(c.Key(key+":owner")).Absolute().Size(0, 0).PassThrough()
	origin := anchor.Bounds()
	ui.Overlay(c.Key(key), func() {
		// Margins participate in attachment fitting. Top/Left offsets run
		// afterward and could push a notice fitted above its owner offscreen.
		panel := ui.Column(c.Key("notice")).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(height-bottom-origin.Y, 0, 0, (width-size)/2-origin.X).Width(size).MaxHeight(max(float32(0), height-bottom-24)).Padding(12, 16).Gap(8).Role(ui.RoleStatus).Label(message).Border(.5, chrome.Color(colors.Border)).Radius(12)
		panel.DrawOver(func(_ *ui.Painter, _ ui.Rect) {
			if anchor.Bounds() != origin {
				c.Services().Invalidate()
			}
		})
		_ = panel.Clicked()
		panel.Cursor(ui.CursorDefault)
		chrome.GlassSurface(panel, chrome.Color(colors.Field), chrome.MaterialPanel, 12)
		panel.Shadow(0, 6, 24, 0, ui.RGBA(0, 0, 0, .18)).Children(func() {
			ui.Scroll(c.Key("notice-message")).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(float32(0), height-bottom-112)).Children(func() {
				ui.Text(c, message).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.FG))
			})
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Justify(ui.End).Children(func() {
				for index, action := range actions {
					if ui.Button(c.Key(fmt.Sprintf("action-%d", index)), action.label).Height(28).Padding(0, 8).BorderWidth(0).Background(ui.Transparent).Disabled(action.disabled).Clicked() && action.run != nil {
						action.run()
					}
				}
				if dismiss != nil && ui.Button(c, paneLabel("关闭提示", "Dismiss")).Height(28).Padding(0, 8).BorderWidth(0).Background(ui.Transparent).Clicked() {
					dismiss()
				}
			})
		})
	})
}
