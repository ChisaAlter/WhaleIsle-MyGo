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
	return dialogSurface(c, backdrop, panel, Color(colors.BG), Color(colors.Border), width, padding)
}

// DialogSurfaceTheme is DialogSurface for callers that hold the ui theme
// rather than the resolved brand colors.
func DialogSurfaceTheme(c *ui.Context, backdrop, panel ui.Element, t *ui.Theme, width, padding float32) ui.Element {
	return dialogSurface(c, backdrop, panel, t.Background, t.Border, width, padding)
}

func dialogSurface(c *ui.Context, backdrop, panel ui.Element, bg, border ui.Color, width, padding float32) ui.Element {
	progress := Reveal(c, backdrop, OverlayDuration)
	mask := ui.RGBA(0, 0, 0, .24)
	luminance := (0.2126*float32(bg.R) + 0.7152*float32(bg.G) + 0.0722*float32(bg.B)) / 255
	if luminance < .5 {
		mask = ui.RGBA(0, 0, 0, .5)
	}
	RevealMask(backdrop, mask, progress)
	panel.Width(width).MaxWidthPercent(94).Padding(padding).Radius(28).Clip().Opacity(progress).
		Background(bg).
		Draw(Glass(bg, MaterialPanel, 28))
	MenuElevation(panel)
	panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if border.A != 0 && r.W > 0 && r.H > 0 {
			p.Stroke(r, border, 28, 0.5)
		}
	})
	return panel
}
