package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// InputSize is coss's input size prop.
type InputSize uint8

const (
	InputMD InputSize = iota // 32 high (30 inside the border)
	InputSM                  // 28
	InputLG                  // 36
)

// inner returns the control's inside height and horizontal padding.
func (s InputSize) inner() (h, padX float32) {
	switch s {
	case InputSM:
		return 26, design.Sp(2.5) - 1
	case InputLG:
		return 34, design.Sp(3) - 1
	}
	return 30, design.Sp(3) - 1
}

// InputOptions configure an Input.
type InputOptions struct {
	Size        InputSize
	Placeholder string
	Disabled    bool
	// Invalid marks the value as not valid (aria-invalid): destructive
	// border and focus ring. Error, when set, also names the problem for
	// assistive technology.
	Invalid  bool
	Error    string
	ReadOnly bool
	Password bool
	// Label names the input for assistive technology when no Field does.
	Label string
	// Unstyled drops the control frame, as inside an InputGroup.
	Unstyled bool
	Key      any
}

// fieldInfo is what an enclosing Field tells the control built in it.
type fieldInfo struct {
	label, description, err string
	disabled, invalid       bool
}

var currentField *fieldInfo

func (o *InputOptions) fromField() {
	if f := currentField; f != nil {
		o.Label = firstNonEmpty(o.Label, f.label)
		o.Error = firstNonEmpty(o.Error, f.err)
		o.Invalid = o.Invalid || f.invalid
		o.Disabled = o.Disabled || f.disabled
	}
}

// controlLook is the shared frame of text controls (input, textarea,
// select trigger, input group, number field): border input, background
// (input/32 in dark), shadow-xs/5 and the ::before highlight at rest;
// border ring and a 3 DIP ring/24 focus ring; destructive/36 border when
// invalid, destructive/64 with a destructive/16 (dark /24) ring when also
// focused.
func controlLook(e ui.Element, t *design.Tokens, s State, radius float32) {
	bg := t.Background
	if t.Glass {
		if t.IsDark() {
			bg = black.Alpha(0.18)
		} else {
			bg = white.Alpha(0.65)
		}
	} else if t.IsDark() {
		bg = t.Input.Alpha(0.32)
	}
	bc := t.Input
	var sh shadowList
	var d deco
	focus := s.Has(StateFocusVisible)
	invalid := s.Has(StateInvalid)
	disabled := s.Has(StateDisabled)
	switch {
	case invalid && focus:
		bc = t.Destructive.Alpha(design.InvalidBorderAlpha)
		sh.add(0, 0, 0.5, design.InputRingWidth, t.Destructive.Alpha(t.InvalidRingAlpha()))
	case invalid:
		bc = t.Destructive.Alpha(0.36)
	case focus:
		bc = t.Ring
		halo := t.Ring.Alpha(design.InputRingAlpha)
		if t.Ring.A < 255 {
			halo = t.Ring.Alpha(0.50)
		}
		sh.add(0, 0, 1, design.InputRingWidth, halo)
	case disabled:
	default:
		if !t.Glass {
			sh.scale(design.ShadowXs, black, 0.05)
			d.highlight(t, 0.04, 0.06)
		}
	}
	e.Background(bg).Border(design.BorderWidth, bc).Radius(radius)
	d.border, d.radius = design.BorderWidth, radius
	finish(e, &sh, &d)
}

// placeholder overlays coss's placeholder (muted-foreground/72, or /75 in
// glass mode for crisp contrast) on an empty text control: the toolkit's own
// takes the theme's muted color.
func placeholder(c *ui.Context, t *design.Tokens, s string, padX, padY, lineHeight float32, ts design.TextStyle) {
	a := float32(design.OpacityPlaceholder)
	if t != nil && t.Glass {
		a = 0.75
	}
	ui.Text(c, s).Absolute().Left(padX).Top(padY).Right(padX).PassThrough().
		FontSize(ts.Size).FixedLineHeight(lineHeight).TextColor(t.MutedForeground.Alpha(a)).SingleLine()
}

// Input is coss's Input: a 32 DIP text field (InputSM 28, InputLG 36)
// editing *value. It returns the text input, whose Changed and Submitted
// report edits.
func Input(c *ui.Context, value *string, o InputOptions) ui.Element {
	withKey(c, o.Key)
	o.fromField()
	t := tok(c)
	h, padX := o.Size.inner()
	var in ui.Element
	control := ui.Row(c).FillWidth().MinWidth(0).Shrink(1)
	control.Children(func() {
		in = ui.TextInputBase(c, value).FillWidth().MinWidth(0).Height(h).PaddingX(padX).
			FontSize(design.TextSM.Size).FixedLineHeight(h).TextColor(t.Foreground).FocusRing(false)
		if o.Password {
			in.Password()
		}
		if o.ReadOnly {
			in.ReadOnly(true)
		}
		if o.Label != "" {
			in.Label(o.Label)
		}
		if o.Error != "" {
			in.Error(o.Error)
		}
		if *value == "" && o.Placeholder != "" {
			in.Placeholder(o.Placeholder)
			placeholder(c, t, o.Placeholder, padX, 0, h, design.TextSM)
		}
	})
	disabled := isDisabled(o.Disabled)
	s := interactField(in, disabled)
	if o.Invalid || o.Error != "" {
		s |= StateInvalid
	}
	if !o.Unstyled {
		controlLook(control, t, s, design.RadiusLG)
		joined(control, design.RadiusLG, ui.Color{})
	}
	disabledLook(control, s)
	return in
}

// TextareaOptions configure a Textarea.
type TextareaOptions struct {
	Size        InputSize
	Placeholder string
	Disabled    bool
	Invalid     bool
	Error       string
	ReadOnly    bool
	Label       string
	// MinLines and MaxLines grow the textarea with its text (field-sizing:
	// content); zero MaxLines grows without a limit.
	MaxLines int
	Unstyled bool
	Key      any
}

// Textarea is coss's Textarea: at least 70 DIPs high (InputSM 66, InputLG
// 74), growing with its text. It returns the text area.
func Textarea(c *ui.Context, value *string, o TextareaOptions) ui.Element {
	withKey(c, o.Key)
	io := InputOptions{Label: o.Label, Error: o.Error, Invalid: o.Invalid, Disabled: o.Disabled}
	io.fromField()
	t := tok(c)
	minH, padY, padX := float32(70), design.Sp(1.5)-1, design.Sp(3)-1
	switch o.Size {
	case InputSM:
		minH, padY, padX = 66, design.Sp(1)-1, design.Sp(2.5)-1
	case InputLG:
		minH, padY = 74, design.Sp(2)-1
	}
	var in ui.Element
	control := ui.Column(c).FillWidth().MinWidth(0)
	control.Children(func() {
		in = ui.TextAreaBase(c, value).FillWidth().MinHeight(minH).Padding(padY, padX).
			FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).TextColor(t.Foreground).FocusRing(false)
		lines := int((minH - 2*padY) / design.TextSM.LineHeight)
		maxLines := o.MaxLines
		if maxLines <= 0 {
			maxLines = 1 << 16
		}
		in.Lines(lines, max(lines, maxLines))
		if o.ReadOnly {
			in.ReadOnly(true)
		}
		if io.Label != "" {
			in.Label(io.Label)
		}
		if io.Error != "" {
			in.Error(io.Error)
		}
		if *value == "" && o.Placeholder != "" {
			in.Placeholder(o.Placeholder)
			placeholder(c, t, o.Placeholder, padX, padY, design.TextSM.LineHeight, design.TextSM)
		}
	})
	s := interactField(in, isDisabled(io.Disabled))
	if io.Invalid || io.Error != "" {
		s |= StateInvalid
	}
	if !o.Unstyled {
		controlLook(control, t, s, design.RadiusLG)
	}
	disabledLook(control, s)
	return in
}
