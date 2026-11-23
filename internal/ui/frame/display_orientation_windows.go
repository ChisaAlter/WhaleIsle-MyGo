//go:build windows

package frame

import "github.com/egoist/mygo/ui"

// displayLandscape follows screen.availWidth/availHeight, not a desktop
// window's aspect: the work area of the monitor showing the window, which
// MyGo reads again as the window moves or the displays change. A Tester has
// no screen, which leaves the orientation unknown.
func displayLandscape(c *ui.Context) (landscape, known bool) {
	return c.DisplayLandscape()
}
