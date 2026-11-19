package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// settingsNotice reports an acknowledgement ("saved", "opened") as a toast
// instead of a line in the page; see widgets.Notice. Failures do not come
// through here: an error is never a toast, so a failed write calls
// settingsAlert where its control was.
func settingsNotice(c *ui.Context, id string, show bool, text func() string) {
	if text == nil {
		widgets.Notice(c, "settings/"+id, show, nil)
		return
	}
	widgets.Notice(c, "settings/"+id, show, func() string { return normalizeSettingsVendorNotice(id, text()) })
}

// settingsAlert shows a failed setting where its control was: the reason and
// the next step sit in the page, and the raw failure stays in the log.
func settingsAlert(c *ui.Context, id string, show bool, text func() string) {
	if !show || text == nil {
		return
	}
	message := normalizeSettingsVendorNotice(id, text())
	if message == "" {
		return
	}
	widgets.Alert(c, widgets.AlertOptions{ID: "settings/" + id, Message: message})
}
