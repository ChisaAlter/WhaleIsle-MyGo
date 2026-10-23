package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// The pick mask lives above Chat, so it cannot change virtual row measurements.
// All rectangles are the last painted, clipped main-Chat boxes in window DIPs.
type inspectorChatPicker struct {
	mask      ui.Handle
	area      ui.Rect
	preview   ui.Rect
	previewID string
	offset    float32
	pressed   bool
	rows      []session.Message
}

// CancelPicking also retires a held press before native blur synthesizes its up.
func (n *InspectorNavigation) CancelPicking() {
	if n == nil {
		return
	}
	n.Picking = false
	n.Picked = nil
	n.Notice = ""
	n.picker.hide()
	n.picker.pressed = false
}

func (p *inspectorChatPicker) hide() {
	p.preview, p.previewID = ui.Rect{}, ""
}

func inspectorChatPickArea(m Model, r *inspectorChatReveal) ui.Rect {
	return inspectorChatIntersect(inspectorChatFlowRect(m, r.viewport), r.flow)
}

func inspectorChatPickGeometry(c *ui.Context, m Model) {
	n := inspectorChatNavigation(m)
	if n == nil {
		return
	}
	q, r := &n.picker, &n.reveal
	area := inspectorChatPickArea(m, r)
	if q.area != area || q.offset != r.offset {
		q.hide()
		q.area, q.offset = area, r.offset
		if n.Picking {
			c.Services().Invalidate()
		}
	}
	if q.previewID != "" && inspectorChatIntersect(r.rows[q.previewID], area) != q.preview {
		q.hide()
	}
	if n.Picking && (area.W <= 0 || area.H <= 0) {
		n.CancelPicking()
		c.Services().Invalidate()
	}
}

func (n *InspectorNavigation) pickAt(x, y float32) (session.Message, ui.Rect, bool) {
	if !n.Picking || !n.picker.area.Contains(x, y) {
		return session.Message{}, ui.Rect{}, false
	}
	// Hit-test only the painted occurrences. Offscreen history never enters
	// this hot path; source metadata is decoded only on primary release.
	for _, node := range n.picker.rows {
		box := inspectorChatIntersect(n.reveal.rows[node.ID], n.picker.area)
		if box.Contains(x, y) {
			return node, box, true
		}
	}
	return session.Message{}, ui.Rect{}, false
}

func inspectorChatPickerOverlay(c *ui.Context, m Model, colors theme.Colors, list *ui.ListState, scroll *ui.ScrollState) {
	n := inspectorChatNavigation(m)
	if n == nil {
		return
	}
	q := &n.picker
	if !n.Picking {
		q.hide()
		q.pressed = false
		return
	}
	services := c.Services()
	c.CaptureKeyDown(func(ev ui.InputEvent) bool {
		if !n.Picking || ev.Key != ui.KeyEscape {
			return false
		}
		n.CancelPicking()
		services.Invalidate()
		return true
	})
	c.ObserveInput(func(ev ui.InputEvent) {
		if !n.Picking {
			return
		}
		switch ev.Kind {
		case ui.InputBlur:
			n.CancelPicking()
			services.Invalidate()
		case ui.InputScroll:
			q.hide()
			services.Invalidate()
		}
	})
	area := q.area
	if area.W <= 0 || area.H <= 0 {
		return
	}
	ui.Overlay(c, func() {
		mask := ui.Box(c.Key("inspector-chat-pick-mask")).Absolute().Left(area.X).Top(area.Y).Size(area.W, area.H).KeepFocus().Cursor(ui.CursorCrosshair).Role(ui.RoleNone).Bind(&q.mask)
		_, _, over := mask.PointerPosition()
		if !over {
			q.hide()
		}
		mask.HandleInput(func(ev ui.InputEvent) bool {
			if !n.Picking {
				return false
			}
			bounds := q.mask.Bounds(c)
			x, y := bounds.X+ev.X, bounds.Y+ev.Y
			switch ev.Kind {
			case ui.InputPointerDown:
				q.pressed = n.Picking && ev.Button == 0
				return true
			case ui.InputPointerMove:
				q.hide()
				if node, box, ok := n.pickAt(x, y); ok {
					q.preview, q.previewID = box, node.ID
				}
				return true
			case ui.InputPointerUp:
				pressed := q.pressed
				q.pressed = false
				if pressed && ev.Button == 0 {
					if node, _, ok := n.pickAt(x, y); ok {
						target := n.PickTarget(node)
						n.Picked = &target
						n.CancelReveal()
					}
				}
				return true
			case ui.InputScroll:
				q.hide()
				if n.Picking && scroll != nil {
					list.StopScrollAnimation()
					// Native wheel deltas are already DIPs. Only the main Chat
					// scrollport is eligible; child code/output views stay inert.
					scroll.X = min(max(float32(0), scroll.X+ev.DX), scroll.MaxX)
					scroll.Y = min(max(float32(0), scroll.Y+ev.DY), scroll.MaxY)
				}
				return true
			}
			return false
		})
		mask.DrawOver(func(p *ui.Painter, _ ui.Rect) {
			if !n.Picking || q.preview.W <= 0 || q.preview.H <= 0 {
				return
			}
			p.Fill(q.preview, chrome.Color(colors.Accent).Alpha(.15), 4)
			outline := ui.Rect{X: q.preview.X - 1, Y: q.preview.Y - 1, W: q.preview.W + 2, H: q.preview.H + 2}
			p.Stroke(outline, chrome.Color(colors.Accent), 5, 1)
		})
	})
}
