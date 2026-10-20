package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// The harness recipes chat uses for plain text actions. Icon-only, chip,
// segmented, menu and toolbar seats keep their own recipes and are deliberately
// not routed through here.
var (
	buttonGhost   = widgets.ButtonGhost
	buttonPrimary = widgets.ButtonPrimary
	buttonOutline = widgets.ButtonOutline
	buttonDanger  = widgets.ButtonDanger
)

// stdButton paints a harness button when the caller carries a palette. Surfaces
// built without one (bare tests, read-only seats) fall back to the framework
// face so a label never disappears.
func stdButton(c *ui.Context, palette theme.Colors, label string, variant widgets.ButtonVariant, disabled bool) ui.Element {
	if palette.FG == "" {
		button := ui.Button(c, label)
		if disabled {
			button = button.Disabled(true)
		}
		return button
	}
	return widgets.Button(c, palette, label, widgets.ButtonSpec{Variant: variant, Size: widgets.ButtonSmall, Disabled: disabled})
}
