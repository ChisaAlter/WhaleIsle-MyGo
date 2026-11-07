package ui

import "runtime"

// Theme holds the colors and metrics widgets use. Change a copy of
// LightTheme or DarkTheme and set it with Context.SetTheme.
type Theme struct {
	Dark bool
	// Background fills the window.
	Background Color
	// Surface is the face of buttons, inputs and other controls;
	// SurfaceHover and SurfacePressed while hovered or pressed.
	Surface        Color
	SurfaceHover   Color
	SurfacePressed Color
	// Border outlines controls.
	Border Color
	// Text and TextMuted color text; TextMuted is for secondary text and
	// placeholders.
	Text      Color
	TextMuted Color
	// Accent colors primary buttons, checked controls and focus rings;
	// AccentText is text on it.
	Accent        Color
	AccentHover   Color
	AccentPressed Color
	AccentText    Color
	Danger        Color
	// Warning and Success color what calls for attention and what went
	// well, as a Meter's levels.
	Warning Color
	Success Color
	// Selection highlights selected text.
	Selection Color
	// Focus is the ring around the control with the keyboard focus.
	Focus Color
	// Inverse fills tooltips and toasts, the theme turned over so that
	// they stand out from what they are over: dark in a light theme, light
	// in a dark one. InverseText is text on it. Left zero, they are Text
	// and Background.
	Inverse     Color
	InverseText Color
	// Scrollbar colors scroll bar thumbs, ScrollbarWidth DIPs wide (6 by
	// default; 0 is 6). ScrollbarHover colors them while the pointer is over
	// the container or a drag is moving them; its zero value keeps Scrollbar.
	Scrollbar      Color
	ScrollbarHover Color
	// ScrollbarAlways keeps scroll bar thumbs painted while the pointer is
	// elsewhere; the zero value paints them on hover and drag only.
	ScrollbarAlways bool
	ScrollbarWidth  float32
	// Radius rounds the corners of controls.
	Radius float32
	// Spacing is the unit of the room widgets leave inside and between
	// their parts: their paddings and gaps, and the sizes of check boxes,
	// switches, sliders and the rows of tables and trees, are multiples
	// of it. 4 by default: 3 makes widgets compact, 5 roomy; 0 is 4.
	Spacing float32
	// FontSize is the size of text, Font its family ("" is the system's).
	FontSize float32
	Font     string
	// Menu is the look of menus, select popups and popovers. Its zero value
	// takes the colors above.
	Menu MenuStyle
}

// MenuStyle is the look of the panels of menus, select popups and
// popovers, and of their rows. A zero color takes the theme's own: Fill is
// Background, Stroke and Separator are Border, Hover is SurfaceHover, Icon
// is Text and Caption is TextMuted.
type MenuStyle struct {
	// Fill fills the panel, over Material when it is set.
	Fill Color
	// Material, if set, paints under Fill, as a backdrop blur that a
	// translucent Fill lets show through.
	Material Material
	// Stroke is the hairline around the panel, half a DIP outside it.
	Stroke Color
	// Hover fills the row the pointer or the arrow keys are on.
	Hover Color
	// Icon colors the items' icons, Caption their shortcuts.
	Icon    Color
	Caption Color
	// Separator colors the lines between groups of items.
	Separator Color
}

// menuStyle returns the theme's menu look with its zero colors filled in.
func (t *Theme) menuStyle() MenuStyle {
	m := t.Menu
	pick := func(c *Color, def Color) {
		if *c == Transparent {
			*c = def
		}
	}
	pick(&m.Fill, t.Background)
	pick(&m.Stroke, t.Border)
	pick(&m.Hover, t.SurfaceHover)
	pick(&m.Icon, t.Text)
	pick(&m.Caption, t.TextMuted)
	pick(&m.Separator, t.Border)
	return m
}

// Space returns n units of the theme's Spacing, to size elements of your
// own in step with the widgets: ui.Column(c).Gap(t.Space(2)).
func (t *Theme) Space(n float32) float32 {
	s := t.Spacing
	if s <= 0 {
		s = 4
	}
	return n * s
}

// inverse returns the fill of tooltips and toasts, and the color of text
// on it.
func (t *Theme) inverse() (fill, text Color) {
	fill, text = t.Inverse, t.InverseText
	if fill == Transparent {
		fill = t.Text
	}
	if text == Transparent {
		text = t.Background
	}
	return fill, text
}

func (t *Theme) scrollbarWidth() float32 {
	if t.ScrollbarWidth <= 0 {
		return 6
	}
	return t.ScrollbarWidth
}

// Rem returns n times the theme's FontSize, as CSS's rem, to size elements
// with the text.
func (t *Theme) Rem(n float32) float32 {
	if t.FontSize <= 0 {
		return n * defaultFontSize()
	}
	return n * t.FontSize
}

func defaultFontSize() float32 {
	if runtime.GOOS == "darwin" {
		return 13
	}
	return 14
}

// LightTheme returns the theme for a light appearance.
func LightTheme() *Theme {
	return &Theme{
		Background:     Hex("#ffffff"),
		Surface:        Hex("#f4f4f5"),
		SurfaceHover:   Hex("#e9e9ec"),
		SurfacePressed: Hex("#dddde1"),
		Border:         Hex("#d9d9de"),
		Text:           Hex("#18181b"),
		TextMuted:      Hex("#71717a"),
		Accent:         Hex("#2563eb"),
		AccentHover:    Hex("#1d4ed8"),
		AccentPressed:  Hex("#1e40af"),
		AccentText:     Hex("#ffffff"),
		Danger:         Hex("#dc2626"),
		Warning:        Hex("#d97706"),
		Success:        Hex("#16a34a"),
		Selection:      RGBA(37, 99, 235, 0.25),
		Focus:          RGBA(37, 99, 235, 0.55),
		Scrollbar:      RGBA(0, 0, 0, 0.32),
		ScrollbarHover: RGBA(0, 0, 0, 0.5),
		Radius:         6,
		Spacing:        4,
		FontSize:       defaultFontSize(),
	}
}

// DarkTheme returns the theme for a dark appearance.
func DarkTheme() *Theme {
	return &Theme{
		Dark:           true,
		Background:     Hex("#18181b"),
		Surface:        Hex("#27272a"),
		SurfaceHover:   Hex("#323236"),
		SurfacePressed: Hex("#3c3c41"),
		Border:         Hex("#3f3f46"),
		Text:           Hex("#f4f4f5"),
		TextMuted:      Hex("#a1a1aa"),
		Accent:         Hex("#3b82f6"),
		AccentHover:    Hex("#60a5fa"),
		AccentPressed:  Hex("#2563eb"),
		AccentText:     Hex("#ffffff"),
		Danger:         Hex("#ef4444"),
		Warning:        Hex("#f59e0b"),
		Success:        Hex("#22c55e"),
		Selection:      RGBA(59, 130, 246, 0.4),
		Focus:          RGBA(96, 165, 250, 0.6),
		Scrollbar:      RGBA(255, 255, 255, 0.35),
		ScrollbarHover: RGBA(255, 255, 255, 0.55),
		Radius:         6,
		Spacing:        4,
		FontSize:       defaultFontSize(),
	}
}
