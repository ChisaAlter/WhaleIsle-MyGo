package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

// PluginInlinePortals paints and routes only explicitly owned original portals.
// Source CSS coordinates map uniformly from the retained native owner row.
func PluginInlinePortals(c *ui.Context, owner *pluginui.Owner, frame *pluginui.MirrorFrame, row pluginui.MirrorOccurrence, dst ui.Rect) {
	if frame == nil || dst.W <= 0 || row.Rect.W <= 0 {
		return
	}
	scale := float64(dst.W) / row.Rect.W
	for _, item := range frame.MirrorPortals() {
		portal := item
		if portal.Handle != row.Handle || !portal.Visible {
			continue
		}
		x := dst.X + float32((portal.Rect.X-row.Rect.X)*scale)
		y := dst.Y + float32((portal.Rect.Y-row.Rect.Y)*scale)
		w, h := float32(portal.Rect.W*scale), float32(portal.Rect.H*scale)
		ui.Overlay(c, func() {
			ui.Box(c.Key("inline-portal:" + portal.Token)).Absolute().Left(x).Top(y).Width(w).Height(h).Focusable().Label(portal.Text).HandleInput(func(event ui.InputEvent) bool { return owner.MirrorPortalPointer(frame, portal, event, w, h) }).Draw(func(p *ui.Painter, r ui.Rect) { owner.PaintMirror(p, r, frame, portal.Occurrence()) })
		})
	}
}
