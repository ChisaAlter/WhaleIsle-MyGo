package projects

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
)

// pButton is the standard outline secondary action button.
func pButton(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled})
}

// pPrimary is the primary filled action button.
func pPrimary(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: disabled})
}

// pGhost is a low-emphasis ghost button (back navigation, clear, close).
func pGhost(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: disabled})
}

// pDanger is a destructive outline button for inline stop/discard/cancel actions.
func pDanger(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonSM, Disabled: disabled})
}

// pDangerSolid is a filled destructive button for dialog confirmations.
func pDangerSolid(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDestructive, Size: coss.ButtonSM, Disabled: disabled})
}

// pTabButton is a selectable tab button (secondary+pressed when active, ghost otherwise).
func pTabButton(c *ui.Context, label string, active, disabled bool) ui.Element {
	variant := coss.ButtonGhost
	if active {
		variant = coss.ButtonSecondary
	}
	return coss.Button(c, label, coss.ButtonOptions{Variant: variant, Size: coss.ButtonSM, Pressed: active, Disabled: disabled})
}

// pFilterButton is a selectable filter/toggle chip (secondary+pressed when active, outline otherwise).
func pFilterButton(c *ui.Context, label string, active, disabled bool) ui.Element {
	variant := coss.ButtonOutline
	if active {
		variant = coss.ButtonSecondary
	}
	return coss.Button(c, label, coss.ButtonOptions{Variant: variant, Size: coss.ButtonSM, Pressed: active, Disabled: disabled})
}

// pListButton is a full-width left-aligned outline button for selectable list entries.
func pListButton(c *ui.Context, label string, disabled bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled}).FillWidth().MinWidth(0).Justify(ui.Start)
}

// pIconButton creates a 28x28 ghost icon button with an accessible label and tooltip.
func pIconButton(c *ui.Context, key, label string, disabled bool, content func()) ui.Element {
	b := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonIconSM,
		Label:    label,
		Disabled: disabled,
		Key:      key,
		Content:  content,
	})
	if label != "" {
		b.Tooltip(label)
	}
	return b
}

// pInput is coss's Input.
func pInput(c *ui.Context, value *string, o coss.InputOptions) ui.Element {
	return coss.Input(c, value, o)
}

// pRowInput wraps coss.Input in a growing flex box so it never collapses inside ui.Row.
func pRowInput(c *ui.Context, value *string, o coss.InputOptions) ui.Element {
	var in ui.Element
	ui.Box(c).Grow(1).MinWidth(0).Children(func() {
		in = coss.Input(c, value, o)
	})
	return in
}

// pTextarea is coss's Textarea.
func pTextarea(c *ui.Context, value *string, o coss.TextareaOptions) ui.Element {
	return coss.Textarea(c, value, o)
}
