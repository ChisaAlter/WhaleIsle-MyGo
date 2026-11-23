package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// FieldOptions configure a Field.
type FieldOptions struct {
	Label       string
	Description string
	// Error shows below the control in destructive-foreground and marks
	// the control invalid. It must be a sentence from internal/errcopy,
	// never a raw error (docs/error-copy.md).
	Error    string
	Disabled bool
	Key      any
}

// Field is coss's Field: a label, the control control builds, and a
// description or error below, 8 DIPs apart. Inputs, textareas and selects
// built in control take the label, description and error for assistive
// technology, and the error as their invalid state.
func Field(c *ui.Context, o FieldOptions, control func()) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	f := ui.Column(c).AlignItems(ui.Start).Gap(design.Sp(2)).FillWidth()
	f.Children(func() {
		if o.Label != "" {
			l := Label(c, o.Label)
			if o.Disabled {
				l.Opacity(design.OpacityDisabled)
			}
		}
		saved := currentField
		currentField = &fieldInfo{label: o.Label, description: o.Description, err: o.Error, disabled: o.Disabled, invalid: o.Error != ""}
		ui.Column(c).FillWidth().AlignItems(ui.Stretch).Children(control)
		currentField = saved
		switch {
		case o.Error != "":
			para(c, o.Error, design.TextXS, design.WeightNormal, t.DestructiveForeground).Role(ui.RoleStatus)
		case o.Description != "":
			para(c, o.Description, design.TextXS, design.WeightNormal, t.MutedForeground)
		}
	})
	return f
}

// Label is coss's Label: 14/16 medium foreground text.
func Label(c *ui.Context, s string) ui.Element {
	t := tok(c)
	return para(c, s, design.TextStyle{Size: design.TextSM.Size, LineHeight: 16}, design.WeightMedium, t.Foreground)
}

// FieldDescription and FieldError are the field's helper texts on their
// own, for layouts Field does not cover.
func FieldDescription(c *ui.Context, s string) ui.Element {
	return para(c, s, design.TextXS, design.WeightNormal, tok(c).MutedForeground)
}

// FieldError shows an inline error sentence (from internal/errcopy).
func FieldError(c *ui.Context, s string) ui.Element {
	return para(c, s, design.TextXS, design.WeightNormal, tok(c).DestructiveForeground).Role(ui.RoleStatus)
}

// Fieldset groups fields under a legend (coss Fieldset: legend 16/24
// semibold, 24 DIPs between fields).
func Fieldset(c *ui.Context, legend string, fn func()) ui.Element {
	t := tok(c)
	return ui.Column(c).FillWidth().Gap(design.Sp(6)).Role(ui.RoleGroup).Label(legend).Children(func() {
		if legend != "" {
			para(c, legend, design.TextBase, design.WeightSemibold, t.Foreground)
		}
		fn()
	})
}
