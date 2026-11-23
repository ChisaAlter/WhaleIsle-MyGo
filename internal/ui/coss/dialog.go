package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// FooterVariant is a dialog footer's look.
type FooterVariant uint8

const (
	// FooterDefault is a muted strip with a top border.
	FooterDefault FooterVariant = iota
	// FooterBare has no strip.
	FooterBare
)

// DialogOptions configure a Dialog, AlertDialog, Sheet or Drawer.
type DialogOptions struct {
	Title, Description string
	// NoClose hides the ghost close button at the top right (Dialog, Sheet).
	NoClose bool
	// Footer builds the actions, laid out right-aligned 8 DIPs apart.
	Footer        func()
	FooterVariant FooterVariant
	// MaxWidth overrides the panel's maximum width (512 for dialogs, 448
	// for sheets).
	MaxWidth float32
	// DismissDisabled keeps Escape and backdrop clicks from closing it.
	DismissDisabled bool
	// Side places a Sheet (SideRight by default) or Drawer (SideBottom).
	Side Side
	Key  any
}

// modalKind picks the frame of a modal panel.
type modalKind uint8

const (
	kindDialog modalKind = iota
	kindAlert
	kindSheet
	kindDrawer
)

// backdropLook is coss's bg-black/32 backdrop (its backdrop-blur-sm is not
// drawn: the toolkit has no backdrop blur outside materials).
func backdropLook(back ui.Element, part string) {
	back.Background(black.Alpha(0.32))
	motion(back, part, 0, 0)
}

// Dialog is coss's Dialog: a 16 DIP-radius panel (at most 512 wide) a
// quarter of the way down the window over a black/32 backdrop, with a
// header (20/20 bold title, muted description), body and footer. Escape
// and backdrop clicks close it unless DismissDisabled. It returns the
// panel, absent while closed.
func Dialog(c *ui.Context, open *bool, o DialogOptions, body func()) ui.Element {
	return modal(c, open, o, body, kindDialog)
}

// AlertDialog is coss's AlertDialog: a Dialog without a close button whose
// backdrop does not dismiss it; give it its actions in Footer.
func AlertDialog(c *ui.Context, open *bool, o DialogOptions, body func()) ui.Element {
	o.NoClose = true
	o.DismissDisabled = true
	return modal(c, open, o, body, kindAlert)
}

// Sheet is coss's Sheet: a full-height panel along the right (or Side)
// edge, at most 448 wide, sliding 32 DIPs in as it fades.
func Sheet(c *ui.Context, open *bool, o DialogOptions, body func()) ui.Element {
	if o.Side == SideBottom && o.Key == nil && o.MaxWidth == 0 {
		o.Side = SideRight
	}
	return modal(c, open, o, body, kindSheet)
}

// Drawer is coss's Drawer: a bottom panel with rounded top corners and a
// grab bar, sliding up over 450 ms along the drawer curve.
func Drawer(c *ui.Context, open *bool, o DialogOptions, body func()) ui.Element {
	return modal(c, open, o, body, kindDrawer)
}

func modal(c *ui.Context, open *bool, o DialogOptions, body func(), kind modalKind) ui.Element {
	t := tok(c)
	withKey(c, o.Key)
	opts := ui.DialogOptions{DismissDisabled: o.DismissDisabled}
	return ui.DialogBaseWithOptions(c, open, opts, func(back, panel ui.Element) {
		ww, wh := c.Size()
		switch kind {
		case kindSheet:
			backdropLook(back, "sheet/sheet-backdrop")
			back.Row().AlignItems(ui.Stretch)
			switch o.Side {
			case SideLeft:
				back.Justify(ui.Start)
			case SideTop, SideBottom:
				back.Column().Justify(ui.Start)
				if o.Side == SideBottom {
					back.Justify(ui.End)
				}
			default:
				back.Justify(ui.End)
			}
		case kindDrawer:
			backdropLook(back, "drawer/drawer-backdrop")
			back.Column().Justify(ui.End).AlignItems(ui.Stretch).Padding(design.Sp(12), 0, 0, 0)
		default:
			part := "dialog/dialog-backdrop"
			if kind == kindAlert {
				part = "alert-dialog/alert-dialog-backdrop"
			}
			backdropLook(back, part)
			// grid-rows-[1fr_auto_3fr] p-4: a quarter of the free height above.
			back.Column().Justify(ui.Start).AlignItems(ui.Center).Padding(design.Sp(4))
			ph := panel.Bounds().H
			if ph > 0 {
				panel.Margin(max(0, (wh-2*design.Sp(4)-ph)/4), 0, 0, 0)
			}
		}
		modalPanel(c, t, panel, o, body, kind, ww)
	})
}

// DialogPanel draws a dialog's panel in place (the gallery's, and
// layouts that host one themselves).
func DialogPanel(c *ui.Context, o DialogOptions, body func()) ui.Element {
	t := tok(c)
	p := ui.Column(c)
	ww, _ := c.Size()
	modalPanel(c, t, p, o, body, kindDialog, ww)
	return p
}

// SheetPanel and DrawerPanel draw those panels in place.
func SheetPanel(c *ui.Context, o DialogOptions, body func()) ui.Element {
	t := tok(c)
	p := ui.Column(c)
	ww, _ := c.Size()
	if o.Side == SideBottom {
		o.Side = SideRight
	}
	modalPanel(c, t, p, o, body, kindSheet, ww)
	return p
}

// DrawerPanel draws a bottom drawer's panel in place.
func DrawerPanel(c *ui.Context, o DialogOptions, body func()) ui.Element {
	t := tok(c)
	p := ui.Column(c)
	ww, _ := c.Size()
	modalPanel(c, t, p, o, body, kindDrawer, ww)
	return p
}

func modalPanel(c *ui.Context, t *design.Tokens, panel ui.Element, o DialogOptions, body func(), kind modalKind, ww float32) {
	radius := design.RadiusXL2
	var sh shadowList
	sh.scale(design.ShadowLg, black, 0.05)
	d := deco{border: design.BorderWidth}
	d.highlight(t, 0.04, 0.06)
	panel.Background(t.Popover).TextColor(t.PopoverForeground)
	footRadius := radius - 1
	switch kind {
	case kindSheet:
		maxW := o.MaxWidth
		if maxW == 0 {
			maxW = 448
		}
		radius, footRadius = 0, 0
		switch o.Side {
		case SideLeft:
			panel.BorderWidth(0, design.BorderWidth, 0, 0)
		case SideTop:
			panel.BorderWidth(0, 0, design.BorderWidth, 0)
		case SideBottom:
			panel.BorderWidth(design.BorderWidth, 0, 0, 0)
		default:
			panel.BorderWidth(0, 0, 0, design.BorderWidth)
		}
		if o.Side == SideLeft || o.Side == SideRight {
			panel.Width(min(maxW, ww-design.Sp(12))).FillHeight()
		} else {
			panel.FillWidth()
		}
		panel.BorderColor(t.Border)
		part := `sheet/sheet-popup[when:side === "right"]`
		switch o.Side {
		case SideLeft:
			part = `sheet/sheet-popup[when:side === "left"]`
		case SideTop:
			part = `sheet/sheet-popup[when:side === "top"]`
		case SideBottom:
			part = `sheet/sheet-popup[when:side === "bottom"]`
		}
		motion(panel, part, 0, 0)
	case kindDrawer:
		panel.FillWidth().BorderWidth(design.BorderWidth, 0, 0, 0).BorderColor(t.Border).Radius(radius, radius, 0, 0)
		footRadius = 0
		if forced == 0 {
			panel.Transition(ui.ElementTransition{Duration: design.DurationDrawer, Ease: design.EaseDrawer, Position: true,
				Enter: &ui.Motion{Y: 400, Opacity: 1}, Exit: &ui.Motion{Y: 400, Opacity: 1}})
		}
	default:
		maxW := o.MaxWidth
		if maxW == 0 {
			maxW = 512
		}
		panel.Width(min(maxW, ww-2*design.Sp(4))).Radius(radius).Border(design.BorderWidth, t.Border)
		part := "dialog/dialog-popup"
		if kind == kindAlert {
			part = "alert-dialog/alert-dialog-popup"
		}
		motion(panel, part, 0, 0)
	}
	if kind != kindDrawer {
		d.radius = radius
	} else {
		d.before = false
	}
	finish(panel, &sh, &d)
	panel.Children(func() {
		if kind == kindDrawer {
			// drawer-bar: a 4×48 input-colored pill in 12 DIPs of padding.
			ui.Row(c).FillWidth().Padding(design.Sp(3)).Justify(ui.Center).Children(func() {
				ui.Box(c).Size(48, 4).Radius(design.RadiusFull).Background(t.Input)
			})
		}
		hasHeader := o.Title != "" || o.Description != ""
		if hasHeader {
			hdr := ui.Column(c).FillWidth().Gap(design.Sp(2)).Padding(design.Sp(6))
			if body != nil {
				hdr.Padding(design.Sp(6), design.Sp(6), design.Sp(3), design.Sp(6))
			}
			hdr.Children(func() {
				if o.Title != "" {
					para(c, o.Title, design.TextStyle{Size: design.TextXL.Size, LineHeight: design.TextXL.Size}, design.WeightBold, t.PopoverForeground).Role(ui.RoleHeading)
				}
				if o.Description != "" {
					para(c, o.Description, design.TextSM, design.WeightNormal, t.MutedForeground)
				}
			})
		}
		if body != nil {
			b := ui.Column(c).FillWidth()
			top, bottom := design.Sp(6), design.Sp(6)
			if hasHeader {
				top = design.Sp(1)
			}
			if o.Footer != nil && o.FooterVariant == FooterBare {
				bottom = design.Sp(1)
			}
			b.Padding(top, design.Sp(6), bottom, design.Sp(6)).Children(body)
		}
		if o.Footer != nil {
			f := ui.Row(c).FillWidth().Justify(ui.End).Gap(design.Sp(2)).PaddingX(design.Sp(6))
			if o.FooterVariant == FooterDefault {
				f.PaddingY(design.Sp(4)).Background(t.Muted.Alpha(0.72)).BorderWidth(design.BorderWidth, 0, 0, 0).BorderColor(t.Border).
					Radius(0, 0, footRadius, footRadius)
			} else {
				f.Padding(design.Sp(4), design.Sp(6), design.Sp(6), design.Sp(6))
			}
			f.Children(o.Footer)
		}
		if !o.NoClose && kind != kindAlert {
			ui.Box(c).Absolute().Right(design.Sp(2)).Top(design.Sp(2)).Children(func() {
				if Button(c, "", ButtonOptions{Variant: ButtonGhost, Size: ButtonIcon, Icon: IconX, Label: "Close"}).Clicked() {
					c.CloseTopDialog()
				}
			})
		}
	})
}
