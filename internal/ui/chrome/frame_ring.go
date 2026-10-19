package chrome

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The original desktop frame-ring is an inside, one physical pixel hairline.
// Caption controls remain an overlay; this adds no content height or padding.
func WindowFramePainter(win *mygo.Window, dark bool) func(*ui.Painter, ui.Rect) {
	visible := runtime.GOOS == "windows" && win != nil && !win.IsMaximized() && !win.IsFullScreen()
	ink := ui.RGBA(0, 0, 0, .1)
	if dark {
		ink = ui.RGBA(255, 255, 255, .12)
	}
	return func(p *ui.Painter, r ui.Rect) {
		if !visible {
			return
		}
		// MyGo strokes inside the supplied edge; inset would shift the ring by
		// another pixel after edge snapping, unlike the original inset:0 seat.
		p.Stroke(r, ink, 20, 1/p.Scale())
	}
}
