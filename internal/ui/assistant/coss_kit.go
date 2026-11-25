package assistant

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
)

// aButton is coss's outline small button for secondary actions in assistant views.
func aButton(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: off})
}

// aPrimary is coss's default small button for primary/confirm actions.
func aPrimary(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: off})
}

// aDanger is coss's destructive small button for confirmed delete/irreversible actions.
func aDanger(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDestructive, Size: coss.ButtonSM, Disabled: off})
}

// aDangerOutline is coss's destructive-outline small button for cancel/discard/delete entry actions.
func aDangerOutline(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonSM, Disabled: off})
}

// aGhost is coss's ghost small button for subtle/utility actions.
func aGhost(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: off})
}

// aChoice is a selectable toggle/tab button: secondary when selected, outline
// otherwise, with Pressed and aria-selected preserved.
func aChoice(c *ui.Context, label string, selected, off bool) ui.Element {
	variant := coss.ButtonOutline
	if selected {
		variant = coss.ButtonSecondary
	}
	return coss.Button(c, label, coss.ButtonOptions{Variant: variant, Size: coss.ButtonSM, Pressed: selected, Disabled: off}).Selected(selected)
}

// aInput is coss's Input.
func aInput(c *ui.Context, value *string, o coss.InputOptions) ui.Element {
	return coss.Input(c, value, o)
}

// aTextarea is coss's Textarea.
func aTextarea(c *ui.Context, value *string, o coss.TextareaOptions) ui.Element {
	return coss.Textarea(c, value, o)
}

// aCheckbox is coss's Checkbox.
func aCheckbox(c *ui.Context, checked *bool, label string, off bool) ui.Element {
	return coss.Checkbox(c, checked, label, coss.CheckOptions{Disabled: off})
}

// aOptions turns plain choice labels into coss select options.
func aOptions(labels []string) []coss.SelectOption {
	out := make([]coss.SelectOption, len(labels))
	for i, l := range labels {
		out[i] = coss.SelectOption{Value: l}
	}
	return out
}

// aSelect is coss's Select over plain labels, the value being the label.
func aSelect(c *ui.Context, value *string, labels []string, label string, off bool) ui.Element {
	return coss.Select(c, value, aOptions(labels), coss.SelectOptions{Label: label, Disabled: off})
}
