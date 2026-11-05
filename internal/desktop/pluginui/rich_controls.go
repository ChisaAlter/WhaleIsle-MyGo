package pluginui

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type richSeat struct {
	Handle string `json:"handle"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	CL     int    `json:"cl"`
	CT     int    `json:"ct"`
	CR     int    `json:"cr"`
	CB     int    `json:"cb"`
}
type richPlacement struct {
	handle       string
	bounds, clip mygo.Rectangle
}
type richLayout struct {
	page     *bridgePage
	revision uint64
	subject  Subject
	bounds   mygo.Rectangle
	holes    []mygo.Rectangle
	seats    []richSeat
}
type richState struct {
	enabled bool
	begun   bool
	frame   uint64
	// declared is the frame in which a seat was built; painted the frame in which
	// a paint pass with a valid visible area ran for it.
	declared, painted uint64
	nextToken         uint64
	token             uint64
	seats             []richPlacement
	pending, applied  *richLayout
	err               string
}

func validRichType(t string) bool {
	switch t {
	case "fragment", "contenteditable", "file", "color", "date", "datetime-local", "time", "month", "week", "canvas", "iframe", "audio", "video", "object", "embed", "img":
		return true
	}
	return false
}

// BeginNativeFrame revokes seats not painted in this build. EndNativeFrame must
// run after all native DrawOver geometry has been collected, including clips.
func (o *Owner) BeginNativeFrame(visible bool) {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.rich.begun = true
	o.rich.enabled = visible
	o.rich.frame++
	o.rich.seats = nil
	o.modelTargets = map[string]bool{}
	o.mu.Unlock()
	if !visible {
		o.Hide()
	}
}
func (o *Owner) RichError() string { o.mu.Lock(); defer o.mu.Unlock(); return o.rich.err }
func (o *Owner) renderRichControl(c *ui.Context, s Snapshot, outlet string, node NativeNode) ui.Element {
	o.mu.Lock()
	o.rich.declared = o.rich.frame
	o.mu.Unlock()
	seat := ui.Box(c).MinWidth(0).Height(float32(node.Height)).Role(ui.RoleGroup).Label(node.Label)
	if node.Inline && node.Width > 0 {
		seat.Width(float32(node.Width)).Shrink(0)
	} else {
		seat.FillWidth()
	}
	seat.DrawOver(func(p *ui.Painter, r ui.Rect) {
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.closed || o.masked || o.incompatible || o.rich.begun && !o.rich.enabled || o.state.Page != s.Page || o.state.Revision != s.Revision || s.Subject == nil || !o.selected.equal(*s.Subject) {
			return
		}
		bounds, clip := richNativeRect(r), richNativeRect(p.NativeSurfaceClip())
		if clip.Width > 0 && clip.Height > 0 {
			// Only a pass that knows what is visible can say a seat is not.
			o.rich.painted = o.rich.frame
		}
		if bounds.Width < 1 || bounds.Height < 1 || richIntersect(bounds, clip).Width < 1 || richIntersect(bounds, clip).Height < 1 {
			return
		}
		for _, prior := range o.rich.seats {
			if prior.handle == node.Handle {
				return
			}
		}
		if len(o.rich.seats) < 128 {
			o.rich.seats = append(o.rich.seats, richPlacement{node.Handle, bounds, clip})
		}
	})
	if message := o.RichError(); message != "" {
		ui.Text(c, message).FillWidth().TextColor(c.Theme().Warning)
	}
	return seat
}

// A source popup stays anchored and non-modal unless its actual owner declares
// aria-modal (or a modal HTML dialog). Outside/Escape are gestures, not a native
// forced close: the original document handlers decide how to respond.
func (o *Owner) renderRichPopup(c *ui.Context, s Snapshot, outlet string, node NativeNode, anchor ui.Element) {
	ui.Overlay(c, func() {
		w, h := c.Size()
		x, y := float32(node.X), float32(node.Y)
		width, height := min(float32(node.Width), w), min(float32(node.Height), h)
		if node.FullScreen {
			x, y, width, height = 0, 0, w, h
		} else {
			r := anchor.Bounds()
			x = max(0, min(w-width, r.X+x))
			y = max(0, min(h-height, r.Y+y))
		}
		popup := ui.Box(c).Absolute().Left(x).Top(y).Width(width).Height(height)
		if node.Modal {
			popup.Modal().Role(ui.RoleDialog)
		} else {
			popup.Role(ui.RolePopup)
		}
		node.Inline = false
		node.Height = float64(height)
		popup.Children(func() { o.renderRichControl(c, s, outlet, node) })
		if !node.FullScreen && popup.PressedOutside() {
			_ = o.nativeInteraction(s, outlet, node.Handle, "outside", "")
		}
		if popup.OverlayShortcut(0, ui.KeyEscape) {
			_ = o.nativeInteraction(s, outlet, node.Handle, "escape", "")
		}
	})
}
func richNativeRect(r ui.Rect) mygo.Rectangle {
	x, y := int(math.Ceil(float64(r.X))), int(math.Ceil(float64(r.Y)))
	return mygo.Rectangle{X: x, Y: y, Width: max(0, int(math.Floor(float64(r.X+r.W)))-x), Height: max(0, int(math.Floor(float64(r.Y+r.H)))-y)}
}
func richIntersect(a, b mygo.Rectangle) mygo.Rectangle {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	return mygo.Rectangle{X: x, Y: y, Width: max(0, min(a.X+a.Width, b.X+b.Width)-x), Height: max(0, min(a.Y+a.Height, b.Y+b.Height)-y)}
}
func richSubtract(a, b mygo.Rectangle) []mygo.Rectangle {
	b = richIntersect(a, b)
	if b.Width <= 0 || b.Height <= 0 {
		return []mygo.Rectangle{a}
	}
	var out []mygo.Rectangle
	for _, r := range []mygo.Rectangle{{X: a.X, Y: a.Y, Width: a.Width, Height: b.Y - a.Y}, {X: a.X, Y: b.Y + b.Height, Width: a.Width, Height: a.Y + a.Height - b.Y - b.Height}, {X: a.X, Y: b.Y, Width: b.X - a.X, Height: b.Height}, {X: b.X + b.Width, Y: b.Y, Width: a.X + a.Width - b.X - b.Width, Height: b.Height}} {
		if r.Width > 0 && r.Height > 0 {
			out = append(out, r)
		}
	}
	return out
}
func richBuildLayout(rows []richPlacement) (mygo.Rectangle, []mygo.Rectangle, []richSeat, bool) {
	if len(rows) == 0 || len(rows) > 128 {
		return mygo.Rectangle{}, nil, nil, false
	}
	b := rows[0].bounds
	for _, row := range rows {
		r := row.bounds
		right, bottom := max(b.X+b.Width, r.X+r.Width), max(b.Y+b.Height, r.Y+r.Height)
		b.X, b.Y = min(b.X, r.X), min(b.Y, r.Y)
		b.Width, b.Height = right-b.X, bottom-b.Y
	}
	if b.Width < 1 || b.Height < 1 || b.Width > 8192 || b.Height > 8192 {
		return b, nil, nil, false
	}
	holes := []mygo.Rectangle{b}
	var seats []richSeat
	for _, row := range rows {
		r := row.bounds
		visible := richIntersect(r, row.clip)
		seats = append(seats, richSeat{Handle: row.handle, X: r.X - b.X, Y: r.Y - b.Y, W: r.Width, H: r.Height, CL: visible.X - r.X, CT: visible.Y - r.Y, CR: r.X + r.Width - visible.X - visible.Width, CB: r.Y + r.Height - visible.Y - visible.Height})
		var next []mygo.Rectangle
		for _, hole := range holes {
			next = append(next, richSubtract(hole, richIntersect(r, row.clip))...)
		}
		holes = next
		if len(holes) > 4096 {
			return b, nil, nil, false
		}
	}
	return b, holes, seats, true
}
func (o *Owner) EndNativeFrame() {
	if o == nil {
		return
	}
	o.flushModelTargets()
	o.flushPublicRowContexts()
	o.mu.Lock()
	if len(o.rich.seats) == 0 && o.rich.declared == o.rich.frame && o.rich.painted != o.rich.frame && !o.closed && !o.masked && !o.incompatible && o.state.Connected && o.state.Subject != nil && o.selected.equal(*o.state.Subject) {
		// A frame paints in several passes and this overlay can run before the content
		// that built the seats. Revoking now would hide the layout the page has just
		// acknowledged and start the whole handshake again.
		o.mu.Unlock()
		return
	}
	h, g := o.host, o.gateway
	if o.closed || o.masked || o.incompatible || o.rich.begun && !o.rich.enabled || !o.state.Connected || o.state.Subject == nil || !o.selected.equal(*o.state.Subject) || len(o.rich.seats) == 0 {
		had := o.rich.pending != nil || o.rich.applied != nil
		o.rich.pending = nil
		o.rich.applied = nil
		o.rich.nextToken++
		o.rich.token = o.rich.nextToken
		token := o.rich.token
		page, rev, subject := o.page, o.state.Revision, o.selected
		o.mu.Unlock()
		if had && h != nil {
			h.SetVisible(false)
			_ = o.sendToPage(page, map[string]any{"type": "richLayout", "revision": rev, "subject": subject, "token": token, "seats": []richSeat{}})
			// The source keeps its body mask even on autonomous node revocation.
			// Only this post-Hide message may release it for hidden mirror use.
			_ = o.sendToPage(page, map[string]any{"type": "richRelease", "token": token})
		}
		return
	}
	bounds, holes, seats, ok := richBuildLayout(o.rich.seats)
	if !ok {
		o.rich.err = "Plugin rich content exceeds the live native surface bounds"
		o.mu.Unlock()
		o.Hide()
		return
	}
	layout := &richLayout{o.page, o.state.Revision, o.selected, bounds, holes, seats}
	if reflect.DeepEqual(o.rich.applied, layout) {
		token := o.rich.token
		o.mu.Unlock()
		o.presentRichLayout(layout, token)
		return
	}
	if reflect.DeepEqual(o.rich.pending, layout) {
		o.mu.Unlock()
		return
	}
	o.rich.nextToken++
	token := o.rich.nextToken
	o.rich.token = token
	o.rich.pending = layout
	o.rich.applied = nil
	o.rich.err = ""
	o.invalidateMirrorLocked()
	var captureDone chan struct{}
	if o.mirror != nil {
		captureDone = o.mirror.captureDone
	}
	o.mu.Unlock()
	if h == nil || g == nil {
		return
	}
	h.SetVisible(false)
	go func() {
		if captureDone != nil {
			select {
			case <-captureDone:
			case <-time.After(5 * time.Second):
				o.richFailure(layout, token, errors.New("original source capture did not yield"))
				return
			}
		}
		o.mu.Lock()
		current := o.rich.pending == layout && o.rich.token == token && !o.closed && !o.masked && o.page == layout.page && o.state.Revision == layout.revision
		o.mu.Unlock()
		if !current {
			return
		}
		if _, err := h.PluginRichSource(g, bounds.Width, bounds.Height); err != nil {
			o.richFailure(layout, token, err)
			return
		}
		err := o.sendToPage(layout.page, map[string]any{"type": "richLayout", "revision": layout.revision, "subject": layout.subject, "token": token, "seats": seats})
		if err != nil {
			o.richFailure(layout, token, err)
			return
		}
		time.AfterFunc(5*time.Second, func() {
			o.richFailure(layout, token, errors.New("original rich-control paint acknowledgement timed out"))
		})
	}()
}
func (o *Owner) richFailure(layout *richLayout, token uint64, err error) {
	o.mu.Lock()
	if o.rich.pending != layout || o.rich.token != token {
		o.mu.Unlock()
		return
	}
	o.rich.pending = nil
	o.rich.err = fmt.Sprintf("Plugin rich content unavailable: %v", err)
	o.mu.Unlock()
	o.Hide()
	o.notify()
}
func (o *Owner) ackRichLayout(page *bridgePage, token, revision uint64, ok bool) {
	o.mu.Lock()
	layout := o.rich.pending
	if layout == nil || token == 0 || o.rich.token != token || o.closed || o.masked || o.incompatible || o.page != page || layout.page != page || revision != layout.revision || o.state.Revision != revision || !o.selected.equal(layout.subject) || o.rich.begun && !o.rich.enabled {
		o.mu.Unlock()
		return
	}
	o.rich.pending = nil
	if ok {
		o.rich.applied = layout
		o.rich.err = ""
	} else {
		o.rich.applied = nil
		o.rich.err = "The browser could not retain this original rich control in a clipped native seat"
	}
	o.mu.Unlock()
	if ok {
		o.presentRichLayout(layout, token)
	} else {
		o.Hide()
	}
	o.notify()
}

// Presentation and native selection/masking are serialized on the UI thread.
// The websocket ack is not a lease to show after a newer owner has revoked it.
func (o *Owner) presentRichLayout(layout *richLayout, token uint64) {
	o.mu.Lock()
	hasHost := o.host != nil
	o.mu.Unlock()
	if !hasHost {
		return
	}
	mygo.RunOnMain(func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.closed || o.masked || o.incompatible || o.rich.applied != layout || o.rich.token != token || o.page != layout.page || o.state.Revision != layout.revision || !o.selected.equal(layout.subject) || o.rich.begun && !o.rich.enabled || o.host == nil {
			return
		}
		if _, err := o.host.PresentPlugin(layout.bounds, layout.bounds, layout.holes, o.gateway); err != nil {
			o.rich.applied = nil
			o.rich.err = err.Error()
			o.host.SetVisible(false)
		}
	})
}
