//go:build darwin

package darwin

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/internal/platform"
)

// Browser seats share their parent's NSWindow. A transparent hole in a separate
// NSWindow cannot transfer targeting to the underlying application window.
// Moving the existing container (not recreating WKWebView) makes AppKit's normal
// sibling hit testing handle both mouse input and native drag destinations.
// All accesses run on the AppKit thread through Window.do/get.
type browserSeat struct {
	owner   *window
	clip    id
	holes   []platform.Rect
	focused bool
}

var browserSeats = map[*window]*browserSeat{}
var browserClips = map[id]*browserSeat{}
var browserClipOnce sync.Once
var browserHitTest func(id, objc.SEL, NSPoint) id
var browserPathCreate func() uintptr
var browserPathAddRect func(uintptr, uintptr, NSRect)
var browserPathRelease func(uintptr)

func registerBrowserClip() {
	browserClipOnce.Do(func() {
		mustDlopen("/System/Library/Frameworks/QuartzCore.framework/QuartzCore")
		purego.RegisterFunc(&browserHitTest, msgSendAddr)
		purego.RegisterFunc(&browserPathCreate, mustDlsym(libCG, "CGPathCreateMutable"))
		purego.RegisterFunc(&browserPathAddRect, mustDlsym(libCG, "CGPathAddRect"))
		purego.RegisterFunc(&browserPathRelease, mustDlsym(libCG, "CGPathRelease"))
		classDef("MyGoBrowserClipView", "NSView", nil, []objc.MethodDef{
			method("hitTest:", func(self id, _ objc.SEL, point NSPoint) id {
				seat := browserClips[self]
				if seat == nil || sendBool(self, "isHidden") {
					return 0
				}
				// NSView.hitTest receives a point in the receiver's superview coordinates.
				local := msgPointFromView(self, sel("convertPoint:fromView:"), point, send(self, "superview"))
				size := msgRect(self, sel("bounds")).Size
				if !browserSeatContains(size.Width, size.Height, local.X, size.Height-local.Y, seat.holes) {
					return 0
				}
				return browserHitTest(seat.owner.view, sel("hitTest:"), local)
			}),
		})
	})
}

// SetBrowserOcclusions is deliberately optional: only a frameless browser child
// of a native surface is supported. Other windows keep their existing behavior.
func (w *window) SetBrowserOcclusions(holes []platform.Rect) bool {
	if w.closed || w.parent == nil || w.parent.closed || w.parent.surface == nil || w.web == 0 || !w.opts.Frameless || w.opts.Modal {
		return false
	}
	withPool(func() {
		seat := browserSeats[w]
		if seat == nil {
			registerBrowserClip()
			bounds, visible := w.Bounds(), w.IsVisible()
			focused := w.IsFocused()
			seat = &browserSeat{owner: w}
			seat.clip = msgInitRect(send(class("MyGoBrowserClipView"), "alloc"), sel("initWithFrame:"), NSRect{})
			browserSeats[w], browserClips[seat.clip] = seat, seat
			send(w.parent.win, "removeChildWindow:", uintptr(w.win))
			send(w.win, "orderOut:", 0)
			// w.view and w.web retain their original owned references. Replace the
			// hidden shell's content view before attaching that same view to the parent.
			empty := msgInitRect(send(class("NSView"), "alloc"), sel("initWithFrame:"), NSRect{})
			send(w.win, "setContentView:", uintptr(empty))
			release(empty)
			send(seat.clip, "addSubview:", uintptr(w.view))
			send(w.parent.view, "addSubview:positioned:relativeTo:", uintptr(seat.clip), 1, 0)
			send(seat.clip, "setHidden:", boolArg(!visible))
			seat.layout(bounds)
			if focused {
				w.Focus()
			}
		}
		seat.holes = append([]platform.Rect(nil), holes...)
		seat.mask()
	})
	return true
}

func (s *browserSeat) layout(r platform.Rect) {
	send(class("CATransaction"), "begin")
	send(class("CATransaction"), "setDisableActions:", 1)
	defer send(class("CATransaction"), "commit")
	parent := s.owner.parent.ContentBounds()
	height := msgRect(s.owner.parent.view, sel("bounds")).Size.Height
	frame := NSRect{Origin: NSPoint{float64(r.X - parent.X), height - float64(r.Y-parent.Y+r.Height)}, Size: NSSize{float64(r.Width), float64(r.Height)}}
	msgSetRect(s.clip, sel("setFrame:"), frame)
	msgSetRect(s.owner.view, sel("setFrame:"), NSRect{Size: frame.Size})
	s.mask()
}
func (s *browserSeat) bounds() platform.Rect {
	parent := s.owner.parent.ContentBounds()
	frame := msgRect(s.clip, sel("frame"))
	height := msgRect(s.owner.parent.view, sel("bounds")).Size.Height
	return platform.Rect{X: parent.X + int(frame.Origin.X), Y: parent.Y + int(height-frame.Origin.Y-frame.Size.Height), Width: int(frame.Size.Width), Height: int(frame.Size.Height)}
}
func (s *browserSeat) mask() {
	send(class("CATransaction"), "begin")
	send(class("CATransaction"), "setDisableActions:", 1)
	defer send(class("CATransaction"), "commit")
	size := msgRect(s.clip, sel("bounds")).Size
	path := browserPathCreate()
	for _, r := range browserSeatVisible(platform.Rect{Width: int(size.Width), Height: int(size.Height)}, s.holes) {
		browserPathAddRect(path, 0, NSRect{Origin: NSPoint{float64(r.X), size.Height - float64(r.Y+r.Height)}, Size: NSSize{float64(r.Width), float64(r.Height)}})
	}
	send(s.clip, "setWantsLayer:", 1)
	layer := send(class("CAShapeLayer"), "layer")
	msgSetRect(layer, sel("setFrame:"), NSRect{Size: size})
	send(layer, "setPath:", path)
	send(send(s.clip, "layer"), "setMask:", uintptr(layer))
	browserPathRelease(path)
}
func (w *window) releaseBrowserSeat() {
	if s := browserSeats[w]; s != nil {
		delete(browserSeats, w)
		delete(browserClips, s.clip)
		send(s.clip, "removeFromSuperview")
		release(s.clip)
	}
}
func (w *window) browserNativeWindow() id {
	if browserSeats[w] != nil {
		return w.parent.win
	}
	return w.win
}

func (w *window) browserHasFocus() bool {
	if !sendBool(w.browserNativeWindow(), "isKeyWindow") {
		return false
	}
	focus := send(w.browserNativeWindow(), "firstResponder")
	return focus == w.web || (focus != 0 && respondsTo(focus, "isDescendantOf:") && sendBool(focus, "isDescendantOf:", uintptr(w.web)))
}
func syncBrowserSeatFocus(parent *window) {
	for child, seat := range browserSeats {
		if child.parent != parent {
			continue
		}
		focused := child.browserHasFocus()
		if focused == seat.focused {
			continue
		}
		seat.focused = focused
		if focused {
			child.h.Focused()
		} else {
			child.resetProductKeys()
			child.h.Blurred()
		}
	}
}
