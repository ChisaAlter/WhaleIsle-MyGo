package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/theme"
)

// The coss recipes chat uses for plain text actions. Icon-only, chip,
// segmented, menu and toolbar seats keep their own recipes and are deliberately
// not routed through here.
var (
	buttonGhost   = coss.ButtonGhost
	buttonPrimary = coss.ButtonDefault
	buttonOutline = coss.ButtonOutline
	buttonDanger  = coss.ButtonDestructive
)

// stdButton paints a coss button at small size.
func stdButton(c *ui.Context, _ theme.Colors, label string, variant coss.ButtonVariant, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: variant, Size: coss.ButtonSM, Disabled: disabled})
}
