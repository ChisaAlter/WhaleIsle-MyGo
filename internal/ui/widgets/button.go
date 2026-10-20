package widgets

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// ButtonVariant is the harness button recipe: what fills and strokes the face.
type ButtonVariant int

const (
	// ButtonPrimary is the monochrome main action (--dsw-alias-button-primary-*).
	ButtonPrimary ButtonVariant = iota
	// ButtonGhost has no face until hovered or pressed.
	ButtonGhost
	// ButtonOutline draws the 0.5 DIP border-l3 stroke over a transparent face.
	ButtonOutline
	// ButtonToolbar fills with --dsw-alias-button-tool-bar-fill, for toolbars
	// floated over content.
	ButtonToolbar
	// ButtonDanger is the destructive fill (--dsw-alias-state-error-primary).
	ButtonDanger
	// ButtonInfo is the blue action (--dsw-alias-button-info-*), used where the
	// original keeps an accent fill rather than the primary monochrome.
	ButtonInfo
)

// ButtonSize is the harness button size: md 36 DIP with 14px text, sm 28 DIP
// with 12px text (ui-primitives Button.module.css).
type ButtonSize int

const (
	ButtonMedium ButtonSize = iota
	ButtonSmall
)

// ButtonSpec is one button's recipe. Disabled is explicit because the face
// dims to 0.4 at paint time and the caller decides before the frame.
type ButtonSpec struct {
	Variant  ButtonVariant
	Size     ButtonSize
	Disabled bool
	// Leading draws an optional glyph before the label, 16 DIP like the
	// harness icon slot.
	Leading func(*ui.Context)
}

// Button creates the shell's standard button. It uses ButtonBase so the look is
// the harness recipe instead of MyGo's theme face: md 36 DIP, 14px/22 text,
// radius 12, 0 14 padding; sm 28 DIP, 12px/18, radius 8, 0 10 padding.
func Button(c *ui.Context, colors theme.Colors, label string, spec ButtonSpec) ui.Element {
	height, padX, radius, fontSize, lineHeight := float32(36), float32(14), float32(12), float32(14), float32(22)
	if spec.Size == ButtonSmall {
		height, padX, radius, fontSize, lineHeight = 28, 10, 8, 12, 18
	}
	base, hover, pressed := ui.Transparent, theme.Color(colors.Hover), theme.Color(colors.Pressed)
	fg := theme.Color(colors.FG)
	border := ui.Color{}
	switch spec.Variant {
	case ButtonPrimary:
		base, hover, pressed = theme.Color(colors.BrandPrimary), theme.Color(colors.BrandPrimaryHover), theme.Color(colors.BrandPrimaryHover)
		fg = theme.Color(colors.ButtonFG)
	case ButtonOutline:
		border = theme.Color(colors.BorderL3)
	case ButtonToolbar:
		base, hover, pressed = theme.Color(colors.ToolbarFill), theme.Color(colors.ToolbarHover), theme.Color(colors.ToolbarHover)
	case ButtonDanger:
		base, hover, pressed = theme.Color(colors.Danger), theme.Color(colors.DangerHover), theme.Color(colors.DangerHover)
		fg = theme.Color(colors.ButtonFG)
	case ButtonInfo:
		base, hover, pressed = theme.Color(colors.ButtonInfoFill), theme.Color(colors.ButtonInfoHover), theme.Color(colors.ButtonInfoHover)
		fg = theme.Color(colors.ButtonFG)
	}
	b := ui.ButtonBase(c).Label(label).Height(height).Padding(0, padX).Gap(4).
		Radius(radius).FontSize(fontSize).FixedLineHeight(lineHeight).FontWeight(500).
		Background(base).TextColor(fg).Shrink(0)
	if border.A > 0 {
		b = b.Border(0.5, border)
	} else {
		// ButtonBase inherits the theme's 1 DIP frame; the harness ghost and
		// filled recipes have no stroke of their own.
		b = b.BorderWidth(0)
	}
	if spec.Disabled {
		b = b.Disabled(true).Opacity(0.4)
	} else if b.Pressed() {
		b = b.Background(pressed)
	} else if b.Hovered() {
		b = b.Background(hover)
	}
	b.Children(func() {
		if spec.Leading != nil {
			spec.Leading(c)
		}
		ui.Text(c, label).FontSize(fontSize).FixedLineHeight(lineHeight).FontWeight(500).TextColor(fg)
	})
	return b
}
