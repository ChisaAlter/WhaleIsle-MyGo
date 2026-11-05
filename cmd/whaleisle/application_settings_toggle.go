package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/deeplink"
)

// Use the actual named modal rather than a stale route flag: a nested
// confirmation cannot grant the settings toggle ownership of its dismissal.
func (s *mainState) toggleApplicationSettings(c *ui.Context) {
	if c.ShortcutModalName() == "settings" {
		c.CloseTopDialog()
		return
	}
	s.selectMainPanel("")
	s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings})
	c.Invalidate()
}
