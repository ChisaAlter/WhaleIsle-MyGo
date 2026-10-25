package widgets

import (
	"github.com/egoist/mygo/ui"
)

// The metrics of a ConfirmDialog: a 400 DIP panel with rounded corners, a
// raised shadow and the buttons along its bottom edge.
const (
	confirmDialogWidth  = 400
	confirmDialogRadius = 28
	confirmButtonRadius = 8
)

// ConfirmDialogOptions describes a ConfirmDialog.
type ConfirmDialogOptions struct {
	// Open shows the dialog while true. A click on the backdrop, Escape, or a
	// button sets it false.
	Open *bool
	// Title is the dialog's heading, as a question about what will happen.
	Title string
	// Message is the body under the title: what the action does, and what
	// cannot be undone.
	Message string
	// ConfirmLabel labels the button that goes ahead, as "Delete". It is the
	// caller's word; no button shows without one.
	ConfirmLabel string
	// CancelLabel labels the button that backs out. Escape does the same; no
	// button shows without one.
	CancelLabel string
	// Danger paints the confirm button in the theme's danger color, for what
	// cannot be taken back.
	Danger bool
	// Label names the dialog for assistive technology instead of Title.
	Label string
}

// ConfirmDialog asks for a confirmation inside the window, in place of a
// system message box: a modal panel over a dimmed backdrop, with the title, the
// message and the two buttons, the confirm one filled in the theme's accent
// color, or its danger color when the action cannot be undone.
//
// It reports whether the user confirmed, in the frame the click happened, so
// that the caller does the work itself:
//
//	if widgets.ConfirmDialog(c, widgets.ConfirmDialogOptions{
//		Open:         &app.confirmDiscard,
//		Title:        labels.DiscardDraft,
//		Message:      labels.DiscardDraftBody,
//		ConfirmLabel: labels.Discard,
//		CancelLabel:  labels.Keep,
//		Danger:       true,
//	}) {
//		app.discardDraft()
//	}
//
// Escape, the backdrop and the cancel button all dismiss it without
// confirming. It follows the theme in light and dark, and follows the
// desktop's reduce-motion setting like every other motion here.
func ConfirmDialog(c *ui.Context, o ConfirmDialogOptions) bool {
	if o.Open == nil || !*o.Open {
		return false
	}
	t := c.Theme()
	mask := float32(0.24)
	if t.Dark {
		mask = 0.5
	}
	confirmed := false
	name := o.Label
	if name == "" {
		name = o.Title
	}
	ui.DialogBase(c, o.Open, func(backdrop, panel ui.Element) {
		// The dialog enters on the overlay token, as every other modal does: the
		// mask at the theme's depth and the panel at the same progress, so a
		// destructive confirmation is not the one surface that snaps on.
		RevealOverlay(c, backdrop, panel, ui.RGBA(0, 0, 0, mask))
		panel.Width(confirmDialogWidth).MaxWidthPercent(92).
			Padding(24, 24, 20, 24).Gap(14).
			Radius(confirmDialogRadius).Background(t.Background).Border(0.5, t.Border).
			Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, 0.24))
		if name != "" {
			panel.Label(name)
		}
		panel.Children(func() {
			if o.Title != "" {
				ui.Text(c, o.Title).FillWidth().FontSize(15).
					LineHeight(24.0 / 15).FontWeight(600).TextColor(t.Text)
			}
			if o.Message != "" {
				ui.Text(c, o.Message).FillWidth().FontSize(13).
					LineHeight(20.0 / 13).TextColor(t.TextMuted)
			}
			first := ui.Local(panel, "confirm-dialog-focus", func() bool { return false })
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Padding(4, 0, 0, 0).Children(func() {
				if o.CancelLabel != "" {
					cancel := confirmDialogButton(c, o.CancelLabel, false, false)
					if !*first {
						// Backing out is where the keyboard starts, so that
						// Enter on a destructive dialog cannot destroy.
						cancel.Focus()
						*first = true
					}
					if cancel.Clicked() {
						*o.Open = false
					}
				}
				if o.ConfirmLabel != "" {
					if confirmDialogButton(c, o.ConfirmLabel, true, o.Danger).Clicked() {
						confirmed = true
						*o.Open = false
					}
				}
			})
		})
	})
	return confirmed
}

// confirmDialogButton builds a button of the dialog: an outline one that backs
// out, or a filled one that goes ahead, in the danger color when danger.
func confirmDialogButton(c *ui.Context, label string, filled, danger bool) ui.Element {
	t := c.Theme()
	button := ui.ButtonBase(c).Padding(7, 14).Radius(confirmButtonRadius).Shrink(0).FocusRing(false).Label(label)
	fill, hover, pressed := t.Surface, t.SurfaceHover, t.SurfacePressed
	text, border := t.Text, t.Border
	if filled {
		fill, hover, pressed = t.Accent, t.AccentHover, t.AccentPressed
		border = ui.Transparent
		if danger {
			fill = t.Danger
			hover = t.Danger.Mix(t.Text, 0.12)
			pressed = t.Danger.Mix(t.Text, 0.24)
		}
		text = readableOn(fill)
	}
	button.TextColor(text)
	// The face eases between its resting, hover and pressed fill over the fast
	// token, as the reference's `transition: background` does. One value runs 0
	// at rest, 1 hovered and 2 pressed, so a press from hover darkens rather than
	// passing back through the resting color.
	target := float32(0)
	switch {
	case button.Pressed():
		target = 2
	case button.Hovered():
		target = 1
	}
	state := HoverMotion.Animate(button, "confirm-face", target)
	hoverShare, pressShare := min(max(state, 0), 1), min(max(state-1, 0), 1)
	button.Draw(func(p *ui.Painter, r ui.Rect) {
		face := fill.Mix(hover, hoverShare).Mix(pressed, pressShare)
		p.Fill(r, face, confirmButtonRadius)
		if border.A > 0 {
			p.Stroke(r, border, confirmButtonRadius, 1)
		}
	})
	button.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if button.FocusVisible() {
			p.FocusRing(r, [4]float32{confirmButtonRadius, confirmButtonRadius, confirmButtonRadius, confirmButtonRadius})
		}
	})
	button.Children(func() {
		ui.Text(c, label).FontSize(13).SingleLine()
	})
	return button
}
