package coss

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// easingOf returns the prebuilt ui.Easing of a coss curve.
func easingOf(c design.CubicBezier) ui.Easing {
	switch c {
	case design.CurveEaseOut:
		return design.EaseOut
	case design.CurveEaseIn:
		return design.EaseIn
	case design.CurveDrawer:
		return design.EaseDrawer
	case design.CurveLinear:
		return design.EaseLinear
	}
	return design.EaseInOut
}

// motion gives a popup part coss's enter and exit: what it animates from as
// it opens (data-starting-style) and to as it closes (data-ending-style),
// over the part's transition. Parts without deltas keep no motion.
func motion(e ui.Element, part string, w, h float32) {
	if forced != 0 {
		return
	}
	in, out := design.EnterOf(part), design.ExitOf(part)
	if in == (design.Delta{}) && out == (design.Delta{}) {
		return
	}
	tr := design.TransitionFor(part)
	et := ui.ElementTransition{Duration: tr.Duration, Ease: easingOf(tr.Curve), Position: true}
	if in != (design.Delta{}) {
		m := in.Motion(w, h)
		et.Enter = &m
	}
	if out != (design.Delta{}) {
		m := out.Motion(w, h)
		et.Exit = &m
	}
	e.Transition(et)
}

// panelLook is the surface of coss's popups (menu, select, popover,
// combobox, tooltip): border, popover fill, shadow-*/5 and the ::before
// highlight. In glass mode it adds frosted backdrop blur, adaptive shadows,
// and a single restrained 0.5 DIP border with no inner rim.
func panelLook(e ui.Element, t *design.Tokens, radius float32, sc design.Shadows) {
	bw := glassBorderWidth(t)
	e.Background(t.Popover).Border(bw, t.Border).Radius(radius).TextColor(t.PopoverForeground)
	var sh shadowList
	sh.scale(sc, black, 0.05)
	d := deco{border: bw, radius: radius}
	d.highlight(t, 0.04, 0.06)
	applyGlassMaterial(e, t, t.Popover, 40, 0.50)
	applyGlassElevation(t, &sh, &d, glassTierPopup)
	finish(e, &sh, &d)
}

// Side places a popup against its anchor.
type Side uint8

const (
	// SideDefault picks the component's default edge (SideBottom for Popover,
	// SideTop for Tooltip, SideRight for Sheet).
	SideDefault Side = iota
	SideBottom
	SideTop
	SideRight
	SideLeft
)

// Align places a popup along its anchor's side.
type Align uint8

const (
	AlignCenter Align = iota
	AlignStart
	AlignEnd
)

// place attaches panel to anchor at side and align, offset DIPs away.
func place(panel, anchor ui.Element, side Side, align Align, offset float32) {
	var at, self ui.Anchor
	switch side {
	case SideDefault, SideBottom, SideTop:
		top := side == SideTop
		pick := func(a, b, c ui.Anchor) ui.Anchor {
			switch align {
			case AlignStart:
				return a
			case AlignEnd:
				return c
			}
			return b
		}
		if top {
			at, self = pick(ui.AnchorTopLeft, ui.AnchorTop, ui.AnchorTopRight), pick(ui.AnchorBottomLeft, ui.AnchorBottom, ui.AnchorBottomRight)
			panel.Margin(0, 0, offset, 0)
		} else {
			at, self = pick(ui.AnchorBottomLeft, ui.AnchorBottom, ui.AnchorBottomRight), pick(ui.AnchorTopLeft, ui.AnchorTop, ui.AnchorTopRight)
			panel.Margin(offset, 0, 0, 0)
		}
	default:
		pick := func(a, b, c ui.Anchor) ui.Anchor {
			switch align {
			case AlignStart:
				return a
			case AlignEnd:
				return c
			}
			return b
		}
		if side == SideRight {
			at, self = pick(ui.AnchorTopRight, ui.AnchorRight, ui.AnchorBottomRight), pick(ui.AnchorTopLeft, ui.AnchorLeft, ui.AnchorBottomLeft)
			panel.Margin(0, 0, 0, offset)
		} else {
			at, self = pick(ui.AnchorTopLeft, ui.AnchorLeft, ui.AnchorBottomLeft), pick(ui.AnchorTopRight, ui.AnchorRight, ui.AnchorBottomRight)
			panel.Margin(0, offset, 0, 0)
		}
	}
	panel.AttachTo(anchor, at, self)
}

// PopoverOptions configure a Popover.
type PopoverOptions struct {
	Side  Side
	Align Align
	// Title and Description head the popover (18/18 semibold, 14/20 muted).
	Title, Description string
	// Width fixes the panel's width; 0 sizes it to its content.
	Width float32
	// TooltipStyle is coss's compact popover: radius 8, 12 px text,
	// shadow-md/5, 4×8 padding.
	TooltipStyle bool
}

// Popover is coss's Popover: a panel below anchor (side bottom, centered,
// 4 DIPs away) while *open, closing on Escape or a press outside. It returns
// the panel, absent while closed.
func Popover(c *ui.Context, anchor ui.Element, open *bool, o PopoverOptions, fn func()) ui.Element {
	t := tok(c)
	return ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
		place(panel, anchor, o.Side, o.Align, 4)
		popoverPanel(c, t, panel, o, fn)
		popoverReveal(c, panel)
	})
}

// PopoverPanel draws a popover's panel in place, without an anchor: for
// layouts that position it themselves, and the gallery.
func PopoverPanel(c *ui.Context, o PopoverOptions, fn func()) ui.Element {
	t := tok(c)
	p := ui.Column(c)
	popoverPanel(c, t, p, o, fn)
	return p
}

func popoverReveal(c *ui.Context, panel ui.Element) ui.Element {
	if rise := popoverRevealRise(c, panel); rise != 0 {
		panel.OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: rise} })
	}
	return panel
}

func popoverRevealRise(c *ui.Context, panel ui.Element) float32 {
	const popoverDuration = 160 * time.Millisecond
	if forced != 0 || c.Preferences().ReduceMotion {
		panel.Opacity(1)
		return 0
	}
	started := ui.Local(panel, "coss.popover.reveal", func() time.Time { return c.Now() })
	elapsed := c.Now().Sub(*started)
	progress := float32(1)
	if elapsed < popoverDuration {
		progress = design.EaseInOut(max(0, float32(elapsed)/float32(popoverDuration)))
		c.After(16 * time.Millisecond)
	}
	if tok(c).Glass {
		panel.Opacity(1)
	} else {
		panel.Opacity(progress)
	}
	if progress < 1 {
		return 4 * (1 - progress)
	}
	return 0
}

func popoverPanel(c *ui.Context, t *design.Tokens, panel ui.Element, o PopoverOptions, fn func()) {
	if o.TooltipStyle {
		panelLook(panel, t, design.RadiusMD, design.ShadowMd)
		panel.Padding(design.Sp(1), design.Sp(2)).FontSize(design.TextXS.Size)
	} else {
		panelLook(panel, t, theme.ContextRadii(c).MD, design.ShadowLg)
		panel.Padding(design.Sp(4))
	}
	if o.Width > 0 {
		panel.Width(o.Width)
	}
	panel.Children(func() {
		if o.Title != "" || o.Description != "" {
			ui.Column(c).Gap(design.Sp(2)).Margin(0, 0, design.Sp(4), 0).Children(func() {
				if o.Title != "" {
					para(c, o.Title, design.TextStyle{Size: design.TextLG.Size, LineHeight: design.TextLG.Size}, design.WeightSemibold, t.PopoverForeground)
				}
				if o.Description != "" {
					para(c, o.Description, design.TextSM, design.WeightNormal, t.MutedForeground)
				}
			})
		}
		if fn != nil {
			fn()
		}
	})
}

// TooltipOptions configure a Tooltip.
type TooltipOptions struct {
	Side  Side
	Align Align
}

// Tooltip is coss's Tooltip: a 12 px label in a popover-colored chip above
// anchor (4 DIPs away) once the pointer rests on it or the keyboard focuses
// it. It returns the tip, absent while hidden.
func Tooltip(c *ui.Context, anchor ui.Element, label string, o TooltipOptions) ui.Element {
	t := tok(c)
	side := o.Side
	if side == SideDefault {
		side = SideTop
	}
	return ui.TooltipBase(c, anchor, func(tip ui.Element) {
		place(tip, anchor, side, o.Align, 4)
		tooltipLook(c, t, tip, label)
	})
}

// TooltipPanel draws a tooltip's chip in place.
func TooltipPanel(c *ui.Context, label string) ui.Element {
	t := tok(c)
	p := ui.Row(c).Shrink(0)
	tooltipLook(c, t, p, label)
	return p
}

func tooltipLook(c *ui.Context, t *design.Tokens, tip ui.Element, label string) {
	panelLook(tip, t, design.RadiusMD, design.ShadowMd)
	tip.Padding(design.Sp(1), design.Sp(2))
	motion(tip, "tooltip/tooltip-popup", 0, 0)
	tip.Children(func() {
		para(c, label, design.TextXS, design.WeightNormal, t.PopoverForeground)
	})
}
