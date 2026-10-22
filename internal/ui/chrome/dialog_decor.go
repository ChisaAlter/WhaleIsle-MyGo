package chrome

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// DialogDecor gives an existing DialogBase panel the modal material without
// changing its layout: a static backdrop mask, glass, elevation and the 0.5px
// hairline. Use DialogSurface when the panel may also take the shared 28px
// metrics; use this when a dialog's own width, height or scrolling must stay
// and its tests click immediately, before a reveal would finish.
func DialogDecor(c *ui.Context, backdrop, panel ui.Element, colors theme.Colors, radius float32) {
	backdrop.Background(ui.RGBA(0, 0, 0, .24))
	panel.Draw(Glass(Color(colors.BG), MaterialPanel, radius))
	MenuElevation(panel)
	panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if border := Color(colors.Border); border.A != 0 && r.W > 0 && r.H > 0 {
			p.Stroke(r, border, radius, 0.5)
		}
	})
}
