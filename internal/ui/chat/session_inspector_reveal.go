package chat

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

const inspectorChatFlashDuration = 700 * time.Millisecond
const inspectorChatScrollQuiet = 150 * time.Millisecond

// The reveal owns a separate window overlay. Its captured rectangle never
// changes transcript layout, and stays fixed for the lifetime of one flash.
type inspectorChatReveal struct {
	binding, session string
	request          *InspectorTarget
	rows             map[string]ui.Rect
	emptyRows        map[string]bool
	rejected         map[string]bool
	viewport         ui.Rect
	flow             ui.Rect
	pending, waiting bool
	locating         bool
	centerQueued     bool
	centerCommitted  bool
	centerList       *ui.ListState
	offset, scale    float32
	lastScroll       float32
	scrollMoved      bool
	quietSince       time.Time
	settledAt        time.Time
	flashAfter       time.Time
	started          time.Time
	rect             ui.Rect
}

// CancelReveal releases an Inspector-owned flash or deferred landing. Cached
// geometry is only used in the same Chat binding and is replaced on every paint.
func (n *InspectorNavigation) CancelReveal() {
	if n == nil {
		return
	}
	n.reveal.cancelCenter()
	n.Target = nil
	n.Highlight = ""
	n.reveal.request = nil
	n.reveal.pending, n.reveal.waiting, n.reveal.locating, n.reveal.centerQueued, n.reveal.centerCommitted = false, false, false, false, false
	n.reveal.started, n.reveal.quietSince, n.reveal.settledAt, n.reveal.flashAfter = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	n.reveal.scrollMoved = false
	n.reveal.rect = ui.Rect{}
	n.reveal.rejected = nil
}

func (r *inspectorChatReveal) cancelCenter() {
	if r.centerList != nil {
		// Stop future writes without queuing an offset that would swallow
		// the wheel or key event currently interrupting the navigation.
		r.centerList.StopScrollAnimation()
		r.centerList = nil
	}
}

func inspectorChatIntersect(a, b ui.Rect) ui.Rect {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	w, h := min(a.X+a.W, b.X+b.W)-x, min(a.Y+a.H, b.Y+b.H)-y
	if w <= 0 || h <= 0 {
		return ui.Rect{}
	}
	return ui.Rect{X: x, Y: y, W: w, H: h}
}

func inspectorChatFlashOpacity(elapsed time.Duration) float32 {
	if elapsed < 0 || elapsed >= inspectorChatFlashDuration {
		return 0
	}
	phase := float32(elapsed) / float32(inspectorChatFlashDuration) * 3
	if phase < 1 {
		return .65 + (.1-.65)*phase
	}
	if phase < 2 {
		return .1 + (.5-.1)*(phase-1)
	}
	return .5 * (3 - phase)
}

func inspectorChatNavigation(m Model) *InspectorNavigation {
	n := m.Inspector
	if n == nil {
		return nil
	}
	if m.DisableDeveloperTools || m.GroupRoom || n.SessionID != m.Snapshot.SelectedID {
		n.CancelPicking()
		n.CancelReveal()
		return nil
	}
	return n
}

func inspectorChatViewport(c *ui.Context, m Model, view ui.Element, list *ui.ListState) {
	n := inspectorChatNavigation(m)
	if n == nil {
		return
	}
	r := &n.reveal
	inspectorChatRevealInput(c, n)
	view.Draw(func(p *ui.Painter, box ui.Rect) {
		r.offset, r.scale = list.ScrollOffset(), p.Scale()
		if r.centerList != nil {
			r.centerCommitted = true
		}
		r.flow = ui.Rect{}
		clear(n.picker.rows)
		n.picker.rows = n.picker.rows[:0]
		if r.rows == nil {
			r.rows = map[string]ui.Rect{}
		} else {
			clear(r.rows)
		}
		if r.emptyRows == nil {
			r.emptyRows = map[string]bool{}
		} else {
			clear(r.emptyRows)
		}
		r.viewport = inspectorChatIntersect(box, p.ClipBounds())
	})
	view.DrawOver(func(p *ui.Painter, _ ui.Rect) {
		inspectorChatPickGeometry(c, m)
		if !r.pending {
			return
		}
		if r.centerQueued {
			return
		}
		if r.emptyRows[n.Highlight] && n.Highlight != "" && r.request != nil {
			if r.rejected == nil {
				r.rejected = map[string]bool{}
			}
			r.rejected[n.Highlight] = true
			r.cancelCenter()
			n.Target = r.request
			c.Services().Invalidate()
			return
		}
		if r.locating {
			r.locating = false
			// Decide from this paint, after virtual rows and viewport have
			// their final geometry. A newly selected owner has no old cache.
			visible := inspectorChatIntersect(r.rows[n.Highlight], r.viewport)
			if visible.W <= 0 || visible.H <= 0 {
				r.centerQueued = true
				c.Services().Invalidate()
				return
			}
		}
		if c.Preferences().ReduceMotion {
			n.CancelReveal()
			return
		}
		if r.waiting {
			if r.quietSince.IsZero() {
				r.quietSince = p.Now()
			}
			if y := list.ScrollOffset(); y != r.lastScroll {
				r.lastScroll, r.quietSince, r.scrollMoved = y, p.Now(), true
				r.settledAt = time.Time{}
			}
			if list.ScrollAnimating() {
				return
			}
			// No-op scrolling has no scrollend. Release it after the same
			// quiet fallback as the browser; actual motion ends explicitly.
			if !r.scrollMoved {
				if d := inspectorChatScrollQuiet - p.Now().Sub(r.quietSince); d > 0 {
					p.After(d)
					return
				}
			}
			if r.scrollMoved {
				if r.settledAt.IsZero() {
					r.settledAt = p.Now()
					p.AnimationFrame()
					return
				}
				if !p.Now().After(r.settledAt) {
					p.AnimationFrame()
					return
				}
			}
			r.centerList, r.waiting = nil, false
			r.flashAfter = p.Now()
			p.AnimationFrame()
			return
		}
		if !r.flashAfter.IsZero() && !p.Now().After(r.flashAfter) {
			p.AnimationFrame()
			return
		}
		box := inspectorChatIntersect(r.rows[n.Highlight], r.viewport)
		if box.W <= 0 || box.H <= 0 {
			box = inspectorChatIntersect(inspectorChatFlowRect(m, r.viewport), r.flow)
		}
		r.pending, r.waiting = false, false
		if box.W <= 0 || box.H <= 0 {
			n.CancelReveal()
			return
		}
		r.rect, r.started = box, p.Now()
	})
}

func inspectorChatFlowRect(m Model, viewport ui.Rect) ui.Rect {
	left, right := float32(16), float32(16)
	width := max(float32(0), viewport.W-left-right)
	if m.FullColumnViewport {
		left, right = 32, 37
		axis := m.ContentWidth
		if axis <= 0 {
			axis = 748
		}
		width = max(float32(0), min(axis, viewport.W-left-right))
		left += max(float32(0), (viewport.W-left-right-width)/2)
	}
	return inspectorChatIntersect(ui.Rect{X: viewport.X + left, Y: viewport.Y, W: width, H: viewport.H}, viewport)
}

func inspectorChatRowGeometry(c *ui.Context, m Model, msg session.Message, root ui.Element, top, right, bottom, left float32) {
	n := inspectorChatNavigation(m)
	if n == nil {
		return
	}
	msgID := msg.ID
	root.DrawOver(func(p *ui.Painter, box ui.Rect) {
		contentW, contentH := box.W-left-right, box.H-top-bottom
		n.reveal.emptyRows[msgID] = contentW <= 0 || contentH <= 0
		flow := inspectorChatIntersect(box, p.ClipBounds())
		if flow.W > 0 && flow.H > 0 {
			if n.reveal.flow.W <= 0 {
				n.reveal.flow = flow
			} else {
				old := n.reveal.flow
				x, y := min(old.X, flow.X), min(old.Y, flow.Y)
				n.reveal.flow = ui.Rect{X: x, Y: y, W: max(old.X+old.W, flow.X+flow.W) - x, H: max(old.Y+old.H, flow.Y+flow.H) - y}
			}
		}
		box = ui.Rect{X: box.X + left, Y: box.Y + top, W: max(float32(0), box.W-left-right), H: max(float32(0), box.H-top-bottom)}
		box = inspectorChatIntersect(box, p.ClipBounds())
		if box.W > 0 && box.H > 0 {
			n.reveal.rows[msgID] = box
			n.picker.rows = append(n.picker.rows, msg)
		}
	})
}

func inspectorChatRevealOverlay(c *ui.Context, m Model, colors theme.Colors) {
	n := inspectorChatNavigation(m)
	if n == nil {
		return
	}
	ui.Overlay(c, func() {
		ui.Box(c.Key("inspector-chat-reveal")).Absolute().Left(0).Top(0).Right(0).Bottom(0).PassThrough().DrawOver(func(p *ui.Painter, _ ui.Rect) {
			r := &n.reveal
			if r.started.IsZero() {
				return
			}
			if c.Preferences().ReduceMotion {
				n.CancelReveal()
				return
			}
			alpha := inspectorChatFlashOpacity(p.Now().Sub(r.started))
			if alpha <= 0 {
				n.CancelReveal()
				return
			}
			p.Fill(r.rect, chrome.Color(colors.BubbleHighlight).Alpha(alpha), 6)
			outline := ui.Rect{X: r.rect.X - 1, Y: r.rect.Y - 1, W: r.rect.W + 2, H: r.rect.H + 2}
			p.Stroke(outline, chrome.Color(colors.Accent).Alpha(alpha), 7, 1)
			p.AnimationFrame()
		})
	})
}
