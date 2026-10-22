package chrome

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// DialogSurface gives a DialogBase panel the shell's modal look: a fading
// backdrop mask, the glass material, a 28px radius, the menu elevation and
// the 0.5px outer hairline. It returns the panel so the caller adds its
// content, as ModalView does for settings.
func DialogSurface(c *ui.Context, backdrop, panel ui.Element, colors theme.Colors, width, padding float32) ui.Element {
	progress := Reveal(c, backdrop, OverlayDuration)
	RevealMask(backdrop, ui.RGBA(0, 0, 0, .24), progress)
	panel.Width(width).MaxWidthPercent(94).Padding(padding).Radius(28).Clip().Opacity(progress).
		Background(Color(colors.BG)).
		Draw(Glass(Color(colors.BG), MaterialPanel, 28))
	MenuElevation(panel)
	panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if border := Color(colors.Border); border.A != 0 && r.W > 0 && r.H > 0 {
			p.Stroke(r, border, 28, 0.5)
		}
	})
	return panel
}
