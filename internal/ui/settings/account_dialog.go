package settings

import "github.com/egoist/mygo/ui"

// Account/model dialogs share bounded content and a resident footer. All busy
// dismissal paths use the same gate, including Escape and the backdrop.
func accountDialog(c *ui.Context, open *bool, title string, width float32, busy bool, content func(), footer ...func()) {
	var actions func()
	if len(footer) > 0 {
		actions = footer[0]
	}
	bodyDialogFooterWidth(c, open, title, width, content, actions, ui.DialogOptions{DismissDisabled: busy})
}
