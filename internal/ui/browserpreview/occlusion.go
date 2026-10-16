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
			holes := make([]mygo.Rectangle, 0, len(rects))
			for _, r := range rects {
				if p := b.mini; p != nil && p.host == tab.Host && p.rendered && r == p.bounds {
					continue
				}
				holes = append(holes, r)
			}
			tab.Host.SetOcclusions(holes)
		}
	}
}

// SetMiniPlayerOcclusions applies overlay order independently of its source dock.
func (b *Controller) SetMiniPlayerOcclusions(rects []mygo.Rectangle) {
	if b.mini != nil && b.mini.rendered {
		b.mini.host.SetOcclusions(rects)
	}
}
