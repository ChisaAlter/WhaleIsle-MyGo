package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

// PluginInlineTurn reuses one immutable source frame for this original completed
// boundary. Native layout clips its ordinary row; browser geometry is never
// inferred from a synthetic transcript ordinal or independently rendered owner.
func PluginInlineTurn(c *ui.Context, owner *pluginui.Owner, frame *pluginui.MirrorFrame, tail pluginui.SessionTurnTail) bool {
	if owner == nil || frame == nil {
		return false
	}
	rendered := false
	for _, occurrence := range frame.Occurrences {
		row := occurrence
		if row.Turn != tail.Turn || row.Seq != tail.Seq || !row.Connected || row.Rect.W <= 0 || row.Rect.H <= 0 {
			continue
		}
		rendered = true
		width, height := float32(row.Rect.W), float32(row.Rect.H)
		name := row.Label
		if name == "" {
			name = row.Entry
		}
		label := fmt.Sprintf("%s · turn %d", name, row.Turn)
		client := owner.MirrorTextClient(frame, row)
		element := ui.Box(c.Key("plugin-inline:" + row.Handle))
		position := ui.Local(element, "position", func() ui.Rect { return ui.Rect{} })
		PluginInlinePortals(c, owner, frame, row, *position)
		element.FillWidth().MinWidth(0).AspectRatio(float32(row.Rect.W / row.Rect.H)).Focusable().Label(label).HandleTextInput(client).
			HandleInput(func(event ui.InputEvent) bool {
				if event.Kind == ui.InputBlur {
					client.HandleInput(event)
				}
				if owner.MirrorPointer(frame, row, event, width, height) {
					return true
				}
				return client.HandleInput(event)
			}).
			Draw(func(p *ui.Painter, dst ui.Rect) {
				width, height = dst.W, dst.H
				*position = dst
				client.SetGeometry(width, height)
				clip := p.NativeSurfaceClip()
				if clip.W > 0 && clip.H > 0 && dst.Y < clip.Y+clip.H && dst.Y+dst.H > clip.Y && dst.W > 0 {
					scale := float64(dst.W) / row.Rect.W
					u := float64(max(dst.Y, clip.Y)-dst.Y) / scale
					v := float64(max(dst.Y, clip.Y))
					owner.NoteMirrorNativeFragment(tail, u, v)
				}
				owner.PaintMirror(p, dst, frame, row)
			})
	}
	return rendered
}
