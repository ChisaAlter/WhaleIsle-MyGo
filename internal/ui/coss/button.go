package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// ButtonVariant is coss's buttonVariants.variant.
type ButtonVariant uint8

const (
	ButtonDefault ButtonVariant = iota
	ButtonOutline
	ButtonSecondary
	ButtonGhost
	ButtonDestructive
	ButtonDestructiveOutline
	ButtonLink
)

// ButtonSize is coss's buttonVariants.size.
type ButtonSize uint8

const (
	ButtonMD ButtonSize = iota // size=default: 32 high
	ButtonXS
	ButtonSM
	ButtonLG
	ButtonXL
	ButtonIconXS
	ButtonIconSM
	ButtonIcon
	ButtonIconLG
	ButtonIconXL
)

// ButtonOptions configure a Button. The zero value is the default variant
// at the default size.
type ButtonOptions struct {
	Variant  ButtonVariant
	Size     ButtonSize
	Disabled bool
	// Loading disables the button and overlays a Spinner; the label keeps
	// its room, transparent.
	Loading bool
	// Icon goes before the label, TrailingIcon after it. Icon sizes use
	// only Icon.
	Icon, TrailingIcon Icon
	// Pressed shows the button as data-pressed (a toggled toolbar button,
	// an open menu trigger).
	Pressed bool
	// Label names an icon-only button for assistive technology.
	Label string
	// Content builds custom children instead of the label and icons.
	Content func()
	// Key gives the button a stable identity among its siblings.
	Key any
}

// ButtonMetrics are a size's geometry, exported for composites (InputGroup
// addons, toolbars) that line up with buttons.
type ButtonMetrics struct {
	Height, Width, PadX, Gap, Radius, IconSize float32
	Text                                       design.TextStyle
}

// MetricsOf returns the desktop geometry of size.
func (s ButtonSize) Metrics() ButtonMetrics {
	m := ButtonMetrics{Height: 32, PadX: 11, Gap: design.Sp(2), Radius: design.RadiusLG, IconSize: design.IconSM, Text: design.TextSM}
	switch s {
	case ButtonXS:
		m.Height, m.PadX, m.Gap, m.Radius, m.IconSize, m.Text = 24, design.Sp(2)-1, design.Sp(1), design.RadiusMD, design.IconXS, design.TextXS
	case ButtonSM:
		m.Height, m.PadX, m.Gap = 28, design.Sp(2.5)-1, design.Sp(1.5)
	case ButtonLG:
		m.Height, m.PadX = 36, design.Sp(3.5)-1
	case ButtonXL:
		m.Height, m.PadX, m.IconSize, m.Text = 40, design.Sp(4)-1, design.IconMD, design.TextBase
	case ButtonIconXS:
		m.Height, m.Width, m.PadX, m.Radius, m.IconSize = 24, 24, 0, design.RadiusMD, design.IconXS
	case ButtonIconSM:
		m.Height, m.Width, m.PadX = 28, 28, 0
	case ButtonIcon:
		m.Height, m.Width, m.PadX = 32, 32, 0
	case ButtonIconLG:
		m.Height, m.Width, m.PadX = 36, 36, 0
	case ButtonIconXL:
		m.Height, m.Width, m.PadX, m.IconSize = 40, 40, 0, design.IconMD
	}
	return m
}

// buttonLook is a variant's resolved colors in one state.
type buttonLook struct {
	bg, border, fg ui.Color
	sh             shadowList
	d              deco
	underline      bool
	hasRing        bool
}

func resolveButton(t *design.Tokens, v ButtonVariant, s State) buttonLook {
	var l buttonLook
	dark := t.IsDark()
	pressed := s.Has(StatePressed)
	disabled := s.Has(StateDisabled)
	hover := s.Has(StateHover)
	focus := s.Has(StateFocusVisible)
	filled := func(fill, fg ui.Color) {
		l.bg, l.border, l.fg = fill, fill, fg
		if hover || pressed {
			l.bg = fill.Alpha(0.9)
		}
		switch {
		case pressed:
			l.sh.inset(0, 1, 0, 0, black.Alpha(0.08))
		case disabled:
		default:
			// not-disabled:inset-shadow white/16 + shadow-xs tinted /24.
			l.sh.inset(0, 1, 0, 0, white.Alpha(0.16))
			l.hasRing = focus
			l.sh.scale(design.ShadowXs, fill, 0.24)
		}
	}
	outline := func(fg ui.Color, hoverBg, hoverBorder ui.Color) {
		l.border, l.fg = t.Input, fg
		l.bg = t.Popover
		if dark {
			l.bg = t.Input.Alpha(0.32)
		}
		if hover || pressed {
			l.bg = hoverBg
			if hoverBorder.A > 0 {
				l.border = hoverBorder
			}
		}
		if !disabled && !pressed {
			l.hasRing = focus
			l.sh.scale(design.ShadowXs, black, 0.05)
			l.d.highlight(t, 0.04, 0.06)
		} else if dark && !disabled {
			l.d.highlight(t, 0, 0.02)
		}
	}
	switch v {
	case ButtonDefault:
		filled(t.Primary, t.PrimaryForeground)
	case ButtonDestructive:
		filled(t.Destructive, white)
	case ButtonOutline:
		hb := t.Accent.Alpha(0.5)
		if dark {
			hb = t.Input.Alpha(0.64)
		}
		outline(t.Foreground, hb, ui.Transparent)
	case ButtonDestructiveOutline:
		outline(t.DestructiveForeground, t.Destructive.Alpha(0.04), t.Destructive.Alpha(0.32))
	case ButtonSecondary:
		l.bg, l.fg = t.Secondary, t.SecondaryForeground
		switch {
		case pressed:
			l.bg = t.Secondary.Alpha(0.8)
		case hover:
			l.bg = t.Secondary.Alpha(0.9)
		}
		l.hasRing = focus
	case ButtonGhost:
		l.fg = t.Foreground
		if hover || pressed {
			l.bg = t.Accent
		}
		l.hasRing = focus
	case ButtonLink:
		l.fg = t.Foreground
		l.underline = hover || pressed
		l.hasRing = focus
	}
	return l
}

// Button creates a coss button. Ask the returned element whether it was
// Clicked.
func Button(c *ui.Context, label string, o ButtonOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	m := o.Size.Metrics()
	disabled := isDisabled(o.Disabled || o.Loading)
	b := ui.ButtonBase(c).Height(m.Height).Radius(m.Radius).Gap(m.Gap).FocusRing(false).Cursor(ui.CursorPointer)
	if m.Width > 0 {
		b.Width(m.Width)
	} else {
		b.PaddingX(m.PadX)
	}
	// A visible label names the button; icon-only ones take o.Label.
	if label == "" || m.Width > 0 {
		if name := firstNonEmpty(o.Label, label); name != "" {
			b.Label(name)
		}
	}
	s := interact(b, disabled)
	if o.Pressed {
		s |= StatePressed
	}
	if o.Loading {
		s |= StateLoading
	}
	l := resolveButton(t, o.Variant, s)
	b.Background(l.bg)
	border(b, l.border, l.bg)
	joined(b, m.Radius, l.border)
	l.d.border, l.d.radius = design.BorderWidth, m.Radius
	l.d.setRing(c, b, t, l.hasRing, t.Ring)
	finish(b, &l.sh, &l.d)
	disabledLook(b, s)
	fg := l.fg
	if o.Loading {
		fg = ui.Transparent
	}
	b.Children(func() {
		if o.Content != nil {
			o.Content()
		} else {
			if o.Icon != IconNone {
				icon(c, o.Icon, m.IconSize, fg, false).MarginX(-2)
			}
			if label != "" && m.Width == 0 {
				tx := text(c, label, m.Text, design.WeightMedium, fg)
				if l.underline {
					tx.Underline()
				}
			}
			if o.TrailingIcon != IconNone {
				icon(c, o.TrailingIcon, m.IconSize, fg, false).MarginX(-2)
			}
		}
		if o.Loading {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().PassThrough().Children(func() {
				Spinner(c, SpinnerOptions{Size: m.IconSize, Color: l.fg})
			})
		}
	})
	return b
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
