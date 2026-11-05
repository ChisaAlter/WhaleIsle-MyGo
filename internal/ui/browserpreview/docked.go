package browserpreview

import (
	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/desktop/webview"
)

func (b *Controller) showHost(h *desktop.Host) {
	if b.mini != nil && b.mini.host == h {
		return
	}
	viewport := b.browserDeviceViewport(h)
	h.SetViewport(viewport)
	if viewport != nil && (viewport.Width < 1 || viewport.Height < 1) {
		b.CancelAction()
		h.SetVisible(false)
		return
	}
	h.SetVisible(true)
}

// ViewDocked mounts the same controller/tabs within the single workspace right
// seat. Leaving the seat hides native children; it never closes pages or PTYs.
func (b *Controller) ViewDocked(c *ui.Context, themeID string, dark bool, onClose func()) {
	b.docked = true
	b.viewSeat(c, themeID, dark, onClose)
}
