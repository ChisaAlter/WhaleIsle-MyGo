package frame

import (
	"github.com/egoist/mygo/ui"
)

// viewportDIP is the window's client area in DIPs, the unit every box in this
// frame is laid out in: ui.Context.Size, which is in DIPs on every surface.
func viewportDIP(c *ui.Context) (width, height float32) {
	return c.Size()
}
