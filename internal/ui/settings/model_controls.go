package settings

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// Small controls of the 0.3.13 model editors: the 12/18 hint under a field, the
// 28 DIP ghost icon buttons of a model row, the text-only "restore defaults"
// button, the outlined "add model" pill and the two-way mode switch.

// modelHint is the 12/18 note under a field. tone picks the ink: "" is the
// tertiary one, "warn" the amber, "success" the green and "error" the red.
func modelHint(c *ui.Context, text, tone string) {
	ink := theme.Color(modelColors(c).Muted)
	switch tone {
	case "warn":
		ink = providerEnvWarnInk
	case "success":
		ink = c.Theme().Success
	case "error":
		ink = c.Theme().Danger
	case "hidden":
		// A blank line: the card keeps its height under the user's hands.
		ink = ui.Transparent
	}
	ui.Text(c, text).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(ink)
}

// modelIconButton is the 28 DIP ghost icon button (16 DIP glyph in the tertiary
// ink, primary ink on hover; the danger one turns red) and reports its click.
// tooltip defaults to the label.
func modelIconButton(c *ui.Context, key, label, tooltip string, icon *ui.SVG, danger, disabled bool) bool {
	if tooltip == "" {
		tooltip = label
	}
	button := ui.Button(c.Key(key), "").Label(label).Tooltip(tooltip).Size(28, 28).Shrink(0).Padding(0).Radius(8).
		BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Disabled(disabled)
	ink := theme.Color(modelColors(c).Muted)
	if button.Hovered() && !disabled {
		if danger {
			ink = c.Theme().Danger
			button.Background(c.Theme().Danger.Alpha(.12))
		} else {
			ink = c.Theme().Text
			button.Background(c.Theme().SurfaceHover)
		}
	}
	button.Children(func() { ui.Icon(c, icon).Size(16, 16).TextColor(ink).PassThrough() })
	return button.Clicked()
}

// modelLabelWithHelp is a field label (12/18 medium, secondary ink) followed by
// the (?) help button when help is not empty.
func modelLabelWithHelp(c *ui.Context, key, label, help string) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontWeight(500).TextColor(theme.Color(modelColors(c).Secondary))
		if help != "" {
			modelIconButton(c, key+"/help", providerCopy("认证说明", "Authentication help"), help, bodyIcon("help"), false, false)
		}
	})
}

// modelLinkPill is the text-only 28 DIP button in the tertiary ink.
func modelLinkPill(c *ui.Context, label string, disabled bool) ui.Element {
	button := ui.Button(c, label).Height(28).Shrink(0).Padding(0, 10).Radius(14).BorderWidth(0).Background(ui.Transparent).
		FontSize(theme.InterfaceSize(12)).TextColor(theme.Color(modelColors(c).Muted)).Disabled(disabled)
	if button.Hovered() && !disabled {
		button.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
	}
	return button
}

// modelPlusPill is the outlined 28 DIP pill with a plus glyph ("add model").
func modelPlusPill(c *ui.Context, label string, disabled bool) ui.Element {
	button := ui.Button(c, "").Label(label).AlignSelf(ui.Start).Height(28).Shrink(0).Padding(0, 12).Gap(6).Radius(14).
		Border(.667, theme.Color(modelColors(c).BorderL3)).Background(ui.Transparent).AlignItems(ui.Center).Disabled(disabled)
	if button.Hovered() && !disabled {
		button.Background(c.Theme().SurfaceHover)
	}
	button.Children(func() {
		ui.Icon(c, bodyIcon("plus")).Size(16, 16).Shrink(0).TextColor(c.Theme().Text).PassThrough()
		ui.Text(c, label).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(c.Theme().Text)
	})
	return button
}

// modelSegmented is the two-way mode switch of the add card: equal 28 DIP
// segments, the active one on the primary ink. It reports the picked index (-1
// when nothing was clicked this frame).
func modelSegmented(c *ui.Context, key string, labels []string, active int, disabled bool) int {
	picked := -1
	ui.Row(c.Key(key)).AlignSelf(ui.Start).Shrink(0).Gap(8).Height(28).Children(func() {
		for index, caption := range labels {
			i, text := index, caption
			segment := ui.Button(c.Key(key+"-"+strconv.Itoa(i)), "").Label(text).Height(28).Padding(0, 14).Radius(8).
				BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center)
			if disabled {
				segment = segment.Disabled(true).Opacity(0.5)
			}
			ink := c.Theme().Text
			if i == active {
				segment.Background(ink)
				ink = c.Theme().Background
			} else if segment.Hovered() && !disabled {
				segment.Background(c.Theme().SurfaceHover)
			}
			segment.Children(func() {
				ui.Text(c, text).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(ink)
			})
			if segment.Clicked() && !disabled {
				picked = i
			}
		}
	})
	return picked
}
