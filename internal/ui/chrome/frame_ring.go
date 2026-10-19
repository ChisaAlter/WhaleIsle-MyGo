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
		width := 1 / p.Scale()
		inset := width / 2
		r.X, r.Y, r.W, r.H = r.X+inset, r.Y+inset, r.W-width, r.H-width
		p.Stroke(r, ink, 20-inset, width)
	}
}
