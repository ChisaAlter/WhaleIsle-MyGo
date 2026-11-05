package projects

import "github.com/egoist/mygo/ui"

func HeaderIcon(c *ui.Context, kind string, size float32, ink ui.Color) {
	projectIcon(c, kind, size, ink, false)
}
