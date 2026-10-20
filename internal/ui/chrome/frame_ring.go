package chrome

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The original desktop frame-ring is an inside, one physical pixel hairline.
// Caption controls remain an overlay; this adds no content height or padding.
// The ink follows the active theme's border (--dsw-alias-border-l2) instead of a
// baked grey, so custom themes keep their own hairline.
func WindowFramePainter(win *mygo.Window, dark bool) func(*ui.Painter, ui.Rect) {
	return func(p *ui.Painter, r ui.Rect) {
		if runtime.GOOS != "windows" || win == nil {
			return
		}
		// A maximized or fullscreen window has no silhouette to anchor: the OS
		// edge owns the border there, exactly like the injected page ring.
		if win.IsMaximized() || win.IsFullScreen() {
			return
		}
		ink := p.Theme().Border
		if ink.A == 0 {
			ink = ui.RGBA(0, 0, 0, .1)
			if dark {
				ink = ui.RGBA(255, 255, 255, .12)
			}
		}
		// MyGo strokes inside the supplied edge; inset would shift the ring by
		// another pixel after edge snapping, unlike the original inset:0 seat.
		p.Stroke(r, ink, 20, 1/p.Scale())
	}
}
