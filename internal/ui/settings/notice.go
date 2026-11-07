package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// settingsNotice reports the outcome of an action (an error, "saved", "opened") as a toast
// instead of a line in the page; see widgets.Notice.
func settingsNotice(c *ui.Context, id, tone string, show bool, text func() string) {
	var wrapped func() string
	if text != nil {
		wrapped = func() string {
			return normalizeSettingsVendorNotice(id, text())
		}
	}
	widgets.Notice(c, "settings/"+id, tone, show, wrapped)
}
