package settings

import "github.com/egoist/mygo/ui"

// ConfirmOptions is the content of one in-app confirmation. The settings domain
// hosts it so every destructive surface can ask in the application's own
// chrome instead of opening an operating-system message box.
type ConfirmOptions struct {
	Title        string
	Body         string
	Detail       string
	ConfirmLabel string
	CancelLabel  string
	// Danger paints the confirm action in the theme's danger color.
	Danger    bool
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
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Padding(8, 0, 0, 0).Children(func() {
				if ui.Button(c, cancelLabel).Clicked() {
					finish(opts.OnCancel)
				}
				confirm := ui.PrimaryButton(c, opts.ConfirmLabel)
				if opts.Danger {
					confirm.TextColor(c.Theme().Danger)
				}
				if confirm.Clicked() {
					finish(opts.OnConfirm)
				}
			})
		})
	})
	if !open {
		finish(opts.OnCancel)
	}
}
