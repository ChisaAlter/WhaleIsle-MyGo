package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// The 0.3.13 Models page is made of three surfaces: a bordered row card per
// provider (the raised layer at 80% with a 0.667 border-l4 stroke, 12 DIP
// corners), a setup card under the row being edited (the selector fill), and
// 36 DIP pill buttons. These helpers carry those recipes for the page.

func modelColors(c *ui.Context) theme.Colors {
	dark := c.Theme().Dark
	return theme.ResolveWithSystem(theme.IDSystem, dark).Active(dark)
}

// modelWell is the raised layer: white in the light half, rgb(35, 35, 36) in the dark one.
func modelWell(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return theme.Color(modelColors(c).Field)
	}
	return ui.RGB(255, 255, 255)
}

func modelInputBorder(c *ui.Context) ui.Color { return theme.Color(modelColors(c).BorderL4) }

func modelRowCard(c *ui.Context, key string, content func()) {
	ui.Column(c.Key(key)).FillWidth().MinWidth(0).Padding(12, 16).Gap(8).Radius(12).
		Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).Children(content)
}

func modelSetupCard(c *ui.Context, key string, content func()) {
	ui.Column(c.Key(key)).FillWidth().MinWidth(0).Padding(16).Gap(16).Radius(12).
		Background(theme.Color(modelColors(c).SelectorFill)).Children(content)
}

// modelFieldLabel is the 12/18 medium label above a field, in the secondary ink.
func modelFieldLabel(c *ui.Context, text string) {
	ui.Text(c, text).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontWeight(500).TextColor(theme.Color(modelColors(c).Secondary))
}

// modelPill is the page's button: 36 DIP / 18 radius, or the 28 DIP / 14 radius
// small one, primary (monochrome fill) or outlined.
func modelPill(c *ui.Context, label string, primary, small, disabled bool) ui.Element {
	colors := modelColors(c)
	height, radius, size, pad := float32(36), float32(18), float32(14), float32(14)
	if small {
		height, radius, size, pad = 28, 14, 12, 10
	}
	button := ui.Button(c, label).Height(height).Shrink(0).Padding(0, pad).Radius(radius).FontSize(theme.InterfaceSize(size)).Disabled(disabled)
	if primary {
		button.Background(theme.Color(colors.BrandPrimary)).TextColor(theme.Color(colors.ButtonFG)).BorderWidth(0)
		if button.Hovered() && !disabled {
			button.Background(theme.Color(colors.BrandPrimaryHover))
		}
		return button
	}
	button.Background(ui.Transparent).Border(.667, theme.Color(colors.BorderL3))
	if button.Hovered() && !disabled {
		button.Background(c.Theme().SurfaceHover)
	}
	return button
}

// modelAddPill is the page's full-width "add" row: the outlined 36 DIP pill with a
// 16 DIP plus before its label.
func modelAddPill(c *ui.Context, label string, disabled bool) ui.Element {
	colors := modelColors(c)
	button := ui.Button(c, "").Label(label).FillWidth().Height(36).Shrink(0).Padding(0, 14).Gap(8).Radius(18).
		Border(.667, theme.Color(colors.BorderL3)).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Disabled(disabled)
	if button.Hovered() && !disabled {
		button.Background(c.Theme().SurfaceHover)
	}
	button.Children(func() {
		ui.Icon(c, bodyIcon("plus")).Size(16, 16).Shrink(0).TextColor(c.Theme().Text).PassThrough()
		ui.Text(c, label).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(c.Theme().Text)
	})
	return button
}

// modelDisclosure is the editor's "custom settings" fold: a hairline, 12 DIP of
// air, a 28 DIP summary (12/500 secondary ink, 16 DIP chevron) and the body 12
// DIP below it.
func modelDisclosure(c *ui.Context, key, title string, body func()) {
	colors := modelColors(c)
	anchor := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	open := ui.Local(anchor, "open", func() bool { return false })
	if settingsSearchReveal(c, key) {
		*open = true
	}
	settingsSearchTarget(c, key, title, anchor)
	secondary := theme.Color(colors.Secondary)
	anchor.Children(func() {
		ui.Box(c).FillWidth().Height(.667).Background(theme.Color(colors.Border))
		ui.Box(c).FillWidth().Height(12)
		toggle := ui.Button(c, "").Label(title).AlignSelf(ui.Start).Height(28).Padding(0, 8, 0, 0).Gap(8).Radius(8).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).FocusRing(false)
		toggle.Children(func() {
			icon := "right"
			if *open {
				icon = "down"
			}
			ui.Icon(c, bodyIcon(icon)).Size(16, 16).Shrink(0).TextColor(secondary).PassThrough()
			ui.Text(c, title).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontWeight(500).TextColor(secondary).SingleLine()
		})
		if toggle.Hovered() {
			toggle.Background(c.Theme().SurfaceHover)
		}
		if toggle.Clicked() {
			*open = !*open
		}
		if *open {
			ui.Box(c).FillWidth().Height(12)
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(body)
		}
	})
}
