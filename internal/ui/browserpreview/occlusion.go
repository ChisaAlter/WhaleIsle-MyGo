package browserpreview

import "github.com/egoist/mygo"

// SetWorkspaceOcclusions affects only this occurrence's docked live hosts.
// Detached OS document/PiP windows have a separate presentation lifetime.
func (b *Controller) SetWorkspaceOcclusions(rects []mygo.Rectangle) {
	if b == nil {
		return
	}
	for _, tab := range b.Tabs {
		if tab != nil && tab.Host != nil {
			tab.Host.SetOcclusions(rects)
		}
	}
}
