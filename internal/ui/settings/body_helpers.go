package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

// bodyCard is a feature card inside a settings page: coss's FramePanel, a
// bordered card surface with the xl radius and 20 DIPs of padding.
func bodyCard(c *ui.Context, m Model, content func()) {
	kSurface(c, ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(4), design.Sp(5))).Children(content)
}

// bodyRosterRow keeps metadata above actions at narrow widths without a fixed label column.
func bodyRosterRow(c *ui.Context, title string, content func(), fg, muted ui.Color) {
	ui.Column(c.Key(title)).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Padding(design.Sp(3), 0).Children(func() {
		kLabel(c, title, tk(c).Foreground)
		content()
	})
}

// bodyDialog follows feature-owned settings editors and keeps the caller's draft
// alive until its explicit cancel/save callback acknowledges the operation.
func bodyDialog(c *ui.Context, open *bool, title string, content func(), options ...ui.DialogOptions) {
	bodyDialogFooter(c, open, title, content, nil, options...)
}

// bodyDialogFooter keeps the editor's actions outside its bounded scroll body.
// Short forms size to their content instead of reserving a blank panel.
func bodyDialogFooter(c *ui.Context, open *bool, title string, content, footer func(), options ...ui.DialogOptions) {
	bodyDialogFooterWidth(c, open, title, design.Sp(96), content, footer, options...)
}

// bodyDialogFooterWidth is coss's Dialog for settings editors: the popover
// panel with the 2xl radius over a black/32 backdrop, a header with the 20 DIP
// bold title and the ghost close button, a bounded scrolling body and the
// muted footer strip holding the actions. width is the panel's maximum width;
// the panel keeps coss's 16 DIP viewport inset.
func bodyDialogFooterWidth(c *ui.Context, open *bool, title string, width float32, content, footer func(), options ...ui.DialogOptions) {
	var dismissal ui.DialogOptions
	if len(options) > 0 {
		dismissal = options[0]
	}
	bodyDialogFrame(c, open, width, func() {
		t := tk(c)
		func() {
			ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Padding(design.Sp(6), design.Sp(3), design.Sp(3), design.Sp(6)).Gap(design.Sp(2)).AlignItems(ui.Start).Children(func() {
				kText(c, title, design.TextStyle{Size: design.TextXL.Size, LineHeight: design.TextXL.Size}, design.WeightBold, t.PopoverForeground).
					Grow(1).MinWidth(0).Role(ui.RoleHeading)
				close := coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonIcon, Icon: coss.IconX, Label: providerCopy("关闭", "Close"), Disabled: dismissal.DismissDisabled})
				close.Margin(-design.Sp(3), 0, 0, 0)
				if close.Clicked() {
					if dismissal.OnDismiss != nil {
						dismissal.OnDismiss()
					} else {
						*open = false
					}
				}
			})
			// The panel sizes to its content, so the scroll keeps a viewport
			// budget instead of Grow: the panel's inset max height minus the
			// 56 DIP header and the 69 DIP footer strip when present.
			_, vh := c.Size()
			bodyMax := max(float32(0), vh-2*design.Sp(4)-design.Sp(14))
			if footer != nil {
				bodyMax = max(0, bodyMax-design.Sp(17)-design.BorderWidth)
			}
			ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).Shrink(1).MaxHeight(bodyMax).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Padding(design.Sp(1), design.Sp(6), design.Sp(6), design.Sp(6)).Children(content)
			})
			if footer != nil {
				ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Padding(design.Sp(4), design.Sp(6)).Wrap().Gap(design.Sp(2)).Justify(ui.End).AlignItems(ui.Center).
					Background(t.Muted.Alpha(0.72)).BorderWidth(design.BorderWidth, 0, 0, 0).BorderColor(t.Border).Children(footer)
			}
		}()
	}, dismissal)
}
