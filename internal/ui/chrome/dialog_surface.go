package chrome

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// DialogSurface gives a DialogBase panel the shell's modal look: a fading
// backdrop mask, the glass material, the 24px dialog radius, modal elevation and
// the 0.5px outer hairline. It returns the panel so the caller adds its
// content, as ModalView does for settings.
func DialogSurface(c *ui.Context, backdrop, panel ui.Element, colors theme.Colors, width, padding float32) ui.Element {
	return dialogSurface(c, backdrop, panel, Color(colors.BG), Color(colors.Border), DialogSurfaceBox{Width: width, Radius: theme.ContextRadii(c).Dialog, Padding: padding})
}

// DialogSurfaceTheme is DialogSurface for callers that hold the ui theme
// rather than the resolved brand colors.
func DialogSurfaceTheme(c *ui.Context, backdrop, panel ui.Element, t *ui.Theme, width, padding float32) ui.Element {
	return dialogSurface(c, backdrop, panel, t.Background, t.Border, DialogSurfaceBox{Width: width, Radius: theme.ContextRadii(c).Dialog, Padding: padding})
}

// DialogSurfaceBox overrides the metrics a feature dialog declares: the width
// and height of the panel, its corner radius, the padding inside the glass, and
// how far its centered place drops down the window. A zero Height keeps the
// panel as tall as its content, as the shared surface does for settings
// editors. A zero OffsetY keeps it centered.
type DialogSurfaceBox struct {
	Width, Height, Radius, Padding, OffsetY float32
}

// DialogSurfaceSized gives a DialogBase panel explicit metrics, for the feature
// dialogs that the installed build sizes itself: the shortcut reference is
// 480x600 with a 24 DIP radius, no inner padding and a 30 DIP drop from the
// centered place, so its header, search row and footer reach the glass edges.
func DialogSurfaceSized(c *ui.Context, backdrop, panel ui.Element, t *ui.Theme, box DialogSurfaceBox) ui.Element {
	if box.Radius <= 0 {
		box.Radius = theme.ContextRadii(c).Dialog
	}
	return dialogSurface(c, backdrop, panel, t.Background, t.Border, box)
}

func dialogMaskColor(bg ui.Color) ui.Color {
	luminance := (0.2126*float32(bg.R) + 0.7152*float32(bg.G) + 0.0722*float32(bg.B)) / 255
	if luminance < .5 {
		return ui.RGBA(0, 0, 0, .5)
	}
	return ui.RGBA(0, 0, 0, .24)
}

func applyDialogBackdropMask(c *ui.Context, backdrop ui.Element, bg ui.Color, progress float32) {
	mask := dialogMaskColor(bg)
	if (backgroundState.active || design.For(c).Glass) && progress > 0 {
		backdrop.Material(MaskBlurMaterial(mask.Alpha(progress), 2*progress))
		return
	}
	RevealMask(backdrop, mask, progress)
}

func applyDialogGlassSurface(c *ui.Context, panel ui.Element, bg, border ui.Color, radius float32) ui.Element {
	dark := c.Theme().Dark
	glassActive := backgroundState.active || design.For(c).Glass
	if glassActive {
		fillBase := MenuBaseFill(theme.Colors{}, dark)
		panel.Background(ui.Transparent).Material(menuPanelGlass{}).Draw(func(p *ui.Painter, r ui.Rect) {
			drawGlass(p, r, fillBase, MaterialMenu, radius, false)
		})
		modalElevationFor(panel, dark)
	} else {
		panel.Background(bg).Draw(Glass(bg, MaterialPanel, radius))
		menuElevationFor(panel, dark)
	}
	panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		if glassActive {
			drawGlassRim(p, r, radius)
		}
		if border.A != 0 {
			p.Stroke(r, border, radius, 0.5)
		}
	})
	return panel
}

func dialogSurface(c *ui.Context, backdrop, panel ui.Element, bg, border ui.Color, box DialogSurfaceBox) ui.Element {
	progress := Reveal(c, backdrop, OverlayDuration)
	applyDialogBackdropMask(c, backdrop, bg, progress)
	panel.Width(box.Width).MaxWidthPercent(94).Padding(box.Padding).Radius(box.Radius).Clip().Opacity(progress)
	if box.Height > 0 {
		panel.Height(box.Height).MaxHeightPercent(94)
	}
	// The backdrop centers the panel; a top margin of twice the drop moves the
	// centered box down by the drop alone, as the reference's translateY does.
	if box.OffsetY != 0 {
		panel.Margin(2*box.OffsetY, 0, 0, 0)
	}
	return applyDialogGlassSurface(c, panel, bg, border, box.Radius)
}
