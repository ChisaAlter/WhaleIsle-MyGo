package chrome

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// DialogDecor gives an existing DialogBase panel the modal material without
// changing its layout: the dim mask, glass, elevation and the 0.5px hairline.
// Use DialogSurface when the panel may also take the shared 28px metrics; use
// this when a dialog's own width, height or scrolling must stay.
func DialogDecor(c *ui.Context, backdrop, panel ui.Element, colors theme.Colors, radius float32) {
	decorateDialog(c, backdrop, panel, Color(colors.BG), Color(colors.Border), radius)
}

// DialogDecorTheme is DialogDecor for callers that hold the ui theme rather
// than the resolved brand colors.
func DialogDecorTheme(c *ui.Context, backdrop, panel ui.Element, t *ui.Theme, radius float32) {
	decorateDialog(c, backdrop, panel, t.Background, t.Border, radius)
}

func decorateDialog(c *ui.Context, backdrop, panel ui.Element, bg, border ui.Color, radius float32) {
	// The mask and the panel enter on the overlay token, as every modal does. It
	// used to be a static fill because a reveal would still be running when a
	// test clicked; opacity only changes the paint, so a click still lands on the
	// first frames (chrome.TestOverlayRevealKeepsControlsClickable).
	RevealOverlay(c, backdrop, panel, ui.RGBA(0, 0, 0, .24))
	panel.Draw(Glass(bg, MaterialPanel, radius))
	MenuElevation(panel)
	panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if border.A != 0 && r.W > 0 && r.H > 0 {
			p.Stroke(r, border, radius, 0.5)
		}
	})
}
