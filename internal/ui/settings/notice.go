package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// settingsNotice reports the outcome of an action (an error, "saved", "opened") as a toast
// instead of a line in the page; see widgets.Notice.
func settingsNotice(c *ui.Context, id, tone string, show bool, text func() string) {
	widgets.Notice(c, "settings/"+id, tone, show, text)
}
