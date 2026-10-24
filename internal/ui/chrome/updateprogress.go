package chrome

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// UpdateProgress is one bounded update operation shown under the title bar:
// checking, download progress, verification and the staged hand-off. It is the
// single-row port of the reference shell update surface
// (reference/electron/src/main/update-dialog.js + update-overlay.js): the Go
// shell owns no separate overlay window, so the same phases paint here and the
// modal confirmation keeps the decision.
type UpdateProgress struct {
	Title string
	// Detail is the bounded evidence line (bytes, version, feed route).
	Detail string
	// Percent is 0..100; a negative value means the length is unknown.
	Percent float64
	// DismissLabel paints the dismissal action; empty hides it. Dismissal only
	// clears this row and never cancels or authorizes the operation.
	DismissLabel string
}

// DrawUpdateProgress paints the progress row when open is true.
func DrawUpdateProgress(c *ui.Context, colors TitleBarColors, open bool, m UpdateProgress, onDismiss func()) {
	if !open {
		return
	}
	muted := colors.Muted
	if muted.A == 0 && muted.R == 0 && muted.G == 0 && muted.B == 0 {
		muted = colors.Foreground
	}
	value := m.Percent / 100
	if m.Percent < 0 {
		value = -1
	} else if value > 1 {
		value = 1
	}
	ui.Column(c).Fill().Background(colors.Background).
		Padding(6, 12, 6, 12).Gap(4).
		Children(func() {
			ui.Row(c).Fill().AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, m.Title).FontSize(theme.InterfaceSize(12)).TextColor(colors.Foreground).Grow(1).MaxLines(2)
				if onDismiss != nil && m.DismissLabel != "" {
					if ui.Button(c, m.DismissLabel).Clicked() {
						onDismiss()
					}
				}
			})
			ui.Progress(c, value).FillWidth()
			if m.Detail != "" {
				ui.Text(c, m.Detail).FontSize(theme.InterfaceSize(11)).TextColor(muted).MaxLines(2)
			}
		})
	ui.Box(c).Fill().Height(1).Background(colors.Border)
}
