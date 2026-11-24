package settings

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
	size := coss.ButtonIconSM
	m := size.Metrics()
	ink := tk(c).Foreground
	if danger {
		ink = tk(c).DestructiveForeground
	}
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     size,
		Label:    label,
		Disabled: disabled,
		Key:      key,
		Content: func() {
			if icon != nil {
				ui.Icon(c, icon).Size(m.IconSize, m.IconSize).Shrink(0).TextColor(ink).Opacity(design.OpacityIcon).PassThrough()
			}
		},
	}).Shrink(0)
	if tooltip != "" {
		button.Tooltip(tooltip)
	}
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
	return kGhost(c, label, disabled).Shrink(0)
}

// modelPlusPill is the outlined 28 DIP pill with a plus glyph ("add model").
func modelPlusPill(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{
		Variant:  coss.ButtonOutline,
		Size:     coss.ButtonSM,
		Icon:     coss.IconPlus,
		Disabled: disabled,
	}).Label(label).AlignSelf(ui.Start).Shrink(0)
}

// modelSegmented is the two-way mode switch of the add card: equal 28 DIP
// segments, the active one on the primary ink. It reports the picked index (-1
// when nothing was clicked this frame).
func modelSegmented(c *ui.Context, key string, labels []string, active int, disabled bool) int {
	picked := -1
	ui.Row(c.Key(key)).AlignSelf(ui.Start).Shrink(0).Gap(design.Sp(2)).Children(func() {
		for index, caption := range labels {
			i, text := index, caption
			segment := kChoiceButton(c.Key(key+"-"+strconv.Itoa(i)), text, i == active, disabled).Label(text)
			if segment.Clicked() && !disabled {
				picked = i
			}
		}
	})
	return picked
}
