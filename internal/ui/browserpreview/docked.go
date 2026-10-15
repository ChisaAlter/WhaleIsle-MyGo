package browserpreview

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/desktop/webview"
)

func (b *Controller) showHost(h *desktop.Host) {
	h.SetViewport(b.viewport)
	if b.viewport != nil && (b.viewport.Width < 1 || b.viewport.Height < 1) {
		b.CancelAction()
		h.SetVisible(false)
		return
	}
	h.SetVisible(true)
}

// ViewDocked mounts the same controller/tabs within the single workspace right
// seat. Leaving the seat hides native children; it never closes pages or PTYs.
func (b *Controller) ViewDocked(c *ui.Context, themeID string, dark bool, onClose func()) {
	box := ui.Column(c).Key("right-browser-seat").Fill().MinWidth(0).MinHeight(0)
	r := box.Bounds()
	b.viewport = &mygo.Rectangle{X: int(r.X), Y: int(r.Y) + BrowserChromeHeight, Width: int(r.W), Height: max(0, int(r.H)-BrowserChromeHeight)}
	box.Children(func() { b.view(c, themeID, dark, onClose) })
}
