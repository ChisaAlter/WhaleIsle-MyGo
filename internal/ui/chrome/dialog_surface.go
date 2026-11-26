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
		return design.TW.Black.Alpha(0.48)
	}
	return design.TW.Black.Alpha(0.24)
}

// dialogGlassBaseFill returns the modal panel tint base for dark and light
// glass surfaces: deep charcoal in dark mode (never washed-out menu gray) and
// clean white in light mode.
func dialogGlassBaseFill(dark bool) ui.Color {
	if dark {
		return ui.RGB(32, 32, 35)
	}
	return design.TW.White
}

type dialogGlassMaterial struct {
	base     ui.Color
	material Material
	blur     bool
}

func (m dialogGlassMaterial) PaintMaterial(p *ui.Painter, r ui.Rect, radii [4]float32) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	if m.blur {
		menuPanelGlass{}.PaintMaterial(p, r, radii)
	}
	drawGlass(p, r, m.base, m.material, radii[0], !m.blur)
}

// ApplyDialogBackdropMask paints the modal dim mask without a fullscreen
// backdrop blur, matching Electron's --dsw-mask-blur: none and avoiding
// full-resolution down=1 GPU blur passes.
func ApplyDialogBackdropMask(_ *ui.Context, backdrop ui.Element, mask ui.Color, progress float32) {
	RevealMask(backdrop, mask, progress)
}

// RevealDialogPanel reveals a modal panel without fading a frosted glass layer's
// own opacity (which breaks backdrop blur sampling), using an 8 DIP upward rise
// in glass mode and opacity fade in non-glass mode.
func RevealDialogPanel(c *ui.Context, panel ui.Element, progress float32) {
	if backgroundState.active || design.For(c).Glass {
		panel.Opacity(1)
		if !c.Preferences().ReduceMotion && progress < 1 {
			rise := 8 * (1 - progress)
			panel.OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: rise} })
		}
		return
	}
	panel.Opacity(progress)
}

func applyDialogGlassSurface(c *ui.Context, panel ui.Element, bg, border ui.Color, _ float32) ui.Element {
	dark := c.Theme().Dark
	t := design.For(c)
	if backgroundState.active {
		fillBase := dialogGlassBaseFill(dark)
		material := MaterialPanel
		if effectivePrefs(backgroundState.prefs).Transparent {
			material = MaterialMenu
		}
		panel.Background(ui.Transparent).Material(dialogGlassMaterial{
			base:     fillBase,
			material: material,
			blur:     true,
		})
		modalElevationFor(panel, dark)
	} else if t.Glass {
		fill := t.Card
		if fill.A == 255 {
			alpha := MaterialAlpha(MaterialPanel, effectivePrefs(backgroundState.prefs), dark)
			if t.Popover.A < 255 {
				alpha = float32(t.Popover.A) / 255
			}
			fill = MaterialFill(dialogGlassBaseFill(dark), alpha)
		}
		panel.Background(ui.Transparent).Material(blurFillMaterial{blur: menuBlur, tint: fill})
		modalElevationFor(panel, dark)
	} else {
		panel.Background(bg).Material(dialogGlassMaterial{
			base:     bg,
			material: MaterialPanel,
			blur:     false,
		})
		menuElevationFor(panel, dark)
	}
	if border.A != 0 {
		panel.Border(0.5, border)
	}
	return panel
}

func dialogSurface(c *ui.Context, backdrop, panel ui.Element, bg, border ui.Color, box DialogSurfaceBox) ui.Element {
	progress := Reveal(c, backdrop, OverlayDuration)
	ApplyDialogBackdropMask(c, backdrop, dialogMaskColor(bg), progress)
	panel.Width(box.Width).MaxWidthPercent(94).Padding(box.Padding).Radius(box.Radius).Clip()
	RevealDialogPanel(c, panel, progress)
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
