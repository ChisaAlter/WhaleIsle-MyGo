package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// ConfirmOptions is the content of one in-app confirmation. The settings domain
// hosts it so every destructive surface can ask in the application's own
// chrome instead of opening an operating-system message box.
type ConfirmOptions struct {
	Title        string
	Body         string
	Detail       string
	ConfirmLabel string
	CancelLabel  string
	// Danger paints the confirm action as the destructive action.
	Danger bool
	// Colors is the active palette; a zero value falls back to the light one so
	// a caller that forgets it still paints readable chrome.
	Colors    theme.Colors
	OnConfirm func()
	OnCancel  func()
}

// ConfirmDialog paints one confirmation modal and reports the answer. It paints
// nothing when called with empty labels, so a caller can render it
// unconditionally while its own state decides whether a question is open.
func ConfirmDialog(c *ui.Context, opts ConfirmOptions) {
	if opts.Title == "" {
		return
	}
	open := true
	settled := false
	finish := func(callback func()) {
		if settled {
			return
		}
		settled = true
		open = false
		if callback != nil {
			callback()
		}
	}
	cancelLabel := opts.CancelLabel
	if cancelLabel == "" {
		cancelLabel = "取消"
	}
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Width(420).MaxWidthPercent(92).Radius(16).Padding(20).Gap(12).Clip().Background(c.Theme().Surface).Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .2)).Children(func() {
			ui.Text(c, opts.Title).FillWidth().FontSize(16).FixedLineHeight(24).FontWeight(600).TextColor(c.Theme().Text)
			if opts.Body != "" {
				ui.Text(c, opts.Body).FillWidth().FontSize(13).FixedLineHeight(20).TextColor(c.Theme().TextMuted)
			}
			if opts.Detail != "" {
				ui.Text(c, opts.Detail).FillWidth().FontSize(12).FixedLineHeight(18).TextColor(c.Theme().TextMuted)
			}
			colors := opts.Colors
			if colors.FG == "" {
				colors = theme.Resolve(theme.IDDeepseekLight).Light
			}
			// A dialog footer is the medium size: cancel is the quiet action and
			// the answer carries the weight (danger when the question is destructive).
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Padding(8, 0, 0, 0).Children(func() {
				if widgets.Button(c, colors, cancelLabel, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonMedium}).Clicked() {
					finish(opts.OnCancel)
				}
				variant := widgets.ButtonPrimary
				if opts.Danger {
					variant = widgets.ButtonDanger
				}
				if widgets.Button(c, colors, opts.ConfirmLabel, widgets.ButtonSpec{Variant: variant, Size: widgets.ButtonMedium}).Clicked() {
					finish(opts.OnConfirm)
				}
			})
		})
	})
	if !open {
		finish(opts.OnCancel)
	}
}
