//go:build !release

package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/egoist/mygo/internal/platform"
)

var (
	windowHostsMu sync.Mutex
	windowHosts   = map[any]*windowHost{}
)

func registerWindowHost(win any, h *windowHost) {
	windowHostsMu.Lock()
	windowHosts[win] = h
	windowHostsMu.Unlock()
}

func unregisterWindowHost(win any) {
	windowHostsMu.Lock()
	delete(windowHosts, win)
	windowHostsMu.Unlock()
}

// WindowDriver drives and inspects the live UI runtime attached to a real
// mygo.Window. All methods must be called on the main thread.
type WindowDriver struct {
	h *windowHost
}

// DriveWindow attaches a WindowDriver to win's live UI runtime, or returns nil
// if win has no attached UI content.
func DriveWindow(win any) *WindowDriver {
	windowHostsMu.Lock()
	h := windowHosts[win]
	windowHostsMu.Unlock()
	if h == nil || h.rt == nil {
		return nil
	}
	h.rt.collect = true
	return &WindowDriver{h: h}
}

func (d *WindowDriver) settle() {
	d.h.rt.collect = true
	for i := 0; i < 20; i++ {
		d.h.requested.Store(false)
		d.h.rt.runFrame()
		if !d.h.requested.Load() {
			return
		}
	}
}

func (d *WindowDriver) send(ev platform.SurfaceEvent) {
	d.h.rt.event(ev)
	d.settle()
}

// Frame renders and settles frames on the live window runtime.
func (d *WindowDriver) Frame() {
	d.h.requested.Store(true)
	d.settle()
}

// Find returns the bounding box of the first element showing text s or labeled s.
func (d *WindowDriver) Find(s string) (Rect, bool) {
	for _, n := range d.h.rt.labels {
		if n.text == s {
			return n.r, true
		}
	}
	return Rect{}, false
}

// FindAll returns all bounding boxes of elements showing text s or labeled s.
func (d *WindowDriver) FindAll(s string) []Rect {
	var out []Rect
	for _, n := range d.h.rt.labels {
		if n.text == s {
			out = append(out, n.r)
		}
	}
	return out
}

// Texts returns the texts of the live window's last frame, in order.
func (d *WindowDriver) Texts() []string {
	var out []string
	for _, n := range d.h.rt.labels {
		out = append(out, n.text)
	}
	return out
}

// HasText reports whether the live window's last frame shows a text containing s.
func (d *WindowDriver) HasText(s string) bool {
	for _, n := range d.h.rt.labels {
		if strings.Contains(n.text, s) {
			return true
		}
	}
	return false
}

// Cursor returns the pointer shape the live window shows.
func (d *WindowDriver) Cursor() Cursor { return d.h.rt.cursor }

// Focused reports whether the element showing text or labeled s, or its
// container, has the keyboard focus in the live window.
func (d *WindowDriver) Focused(s string) bool {
	for _, n := range d.h.rt.labels {
		if n.text != s {
			continue
		}
		for id := n.id; id != 0; {
			if id == d.h.rt.focused {
				return true
			}
			st := d.h.rt.states[id]
			if st == nil {
				break
			}
			id = st.parent
		}
	}
	return false
}

// Move moves the pointer to (x, y) in the live window.
func (d *WindowDriver) Move(x, y float32) {
	d.send(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(x), Y: float64(y)})
}

// Hover moves the pointer to the center of the element showing text or labeled s.
func (d *WindowDriver) Hover(s string) error {
	r, ok := d.Find(s)
	if !ok {
		return fmt.Errorf("ui: no element shows %q", s)
	}
	d.Move(r.X+r.W/2, r.Y+r.H/2)
	return nil
}

// ClickAt clicks at (x, y) in DIPs on the live window.
func (d *WindowDriver) ClickAt(x, y float32) {
	d.ClickAtWith(0, x, y)
}

// ClickAtWith clicks with modifiers, including Shift to extend a selection.
func (d *WindowDriver) ClickAtWith(mods Modifiers, x, y float32) {
	d.send(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(x), Y: float64(y)})
	d.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y), Mods: platform.Modifiers(mods)})
	d.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y), Mods: platform.Modifiers(mods)})
}

// SelectablePoint returns the live layout position of a rune offset in a
// mounted selectable paragraph. It does not construct an offscreen paragraph.
// Callers must ensure the returned point is inside the visible content area.
func (d *WindowDriver) SelectablePoint(text string, at int) (x, y float32, ok bool) {
	for _, s := range d.h.rt.texts {
		if s.editor.source == text {
			cx, cy, h := s.editor.layout.Caret(at)
			return s.x + s.editor.originX + cx, s.y + s.editor.originY + cy + h/2, true
		}
	}
	return 0, 0, false
}

// Click clicks the center of the element showing text or labeled s on the live window.
func (d *WindowDriver) Click(s string) error {
	r, ok := d.Find(s)
	if !ok {
		return fmt.Errorf("ui: no element shows %q", s)
	}
	d.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	return nil
}

// Scroll scrolls by dx, dy DIPs with the pointer at (x, y) on the live window.
func (d *WindowDriver) Scroll(x, y, dx, dy float32) {
	d.send(platform.SurfaceEvent{Kind: platform.PointerScroll, X: float64(x), Y: float64(y), DX: float64(dx), DY: float64(dy), Precise: true})
}

// Key presses a key with modifiers on the live window.
func (d *WindowDriver) Key(mods Modifiers, key Key) {
	d.send(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(key), Mods: platform.Modifiers(mods)})
	d.send(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(key), Mods: platform.Modifiers(mods)})
}

// PhysicalKey sends one physical press/release through the live native-input
// resolver. It never injects OS keyboard input into another window.
func (d *WindowDriver) PhysicalKey(code string, key Key, mods Modifiers) {
	d.send(platform.SurfaceEvent{Kind: platform.KeyPressed, Code: code, Key: platform.Key(key), Mods: platform.Modifiers(mods)})
	d.send(platform.SurfaceEvent{Kind: platform.KeyReleased, Code: code, Key: platform.Key(key), Mods: platform.Modifiers(mods)})
}

// Type types text into the focused text input on the live window.
func (d *WindowDriver) Type(s string) {
	d.send(platform.SurfaceEvent{Kind: platform.TextInput, Text: s})
}
