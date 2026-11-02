package mygo

import (
	"context"
	"errors"
	"sync"

	"github.com/egoist/mygo/internal/platform"
)

type BrowserMouseKind = platform.BrowserMouseKind
type BrowserMouseButton = platform.BrowserMouseButton
type BrowserMouseEvent = platform.BrowserMouseEvent

const (
	BrowserMouseMove    = platform.BrowserMouseMove
	BrowserMousePress   = platform.BrowserMousePress
	BrowserMouseRelease = platform.BrowserMouseRelease
	BrowserMouseWheel   = platform.BrowserMouseWheel
	BrowserMouseCancel  = platform.BrowserMouseCancel
	BrowserMouseNone    = platform.BrowserMouseNone
	BrowserMouseLeft    = platform.BrowserMouseLeft
	BrowserMouseMiddle  = platform.BrowserMouseMiddle
	BrowserMouseRight   = platform.BrowserMouseRight
	BrowserMouseBack    = platform.BrowserMouseBack
	BrowserMouseForward = platform.BrowserMouseForward
)

var ErrBrowserMouseInputUnsupported = errors.New("mygo: typed mouse input is unsupported on this page backend")
var ErrBrowserMouseInputBusy = errors.New("mygo: await the previous mouse input before sending another")

type browserMouseState struct {
	document context.Context
	held     uint8
	busy     bool
}

// Access is confined to the UI thread. Closed removes its only strong key;
// changing document discards held state rather than releasing into the new one.
var browserMouseStates = map[*Window]*browserMouseState{}

// BrowserMouseInputAvailable reports an optional native renderer input provider.
// Windows WebView2 currently implements it. No JavaScript binding or DevTools
// UI is enabled by this query. Host code chooses the source page and whether a
// native interaction is allowed; this API grants no clipboard or permissions.
func (p *Page) BrowserMouseInputAvailable() bool {
	return onMainValue(func() bool {
		return p != nil && p.w != nil && p.w.native != nil && p.w.content == nil && platform.BrowserMouseInputProvider(p.w.native) != nil
	})
}

// MouseInput sends one typed native mouse operation to this page. Coordinates,
// wheel deltas and button masks are defined on BrowserMouseEvent; callers map
// bitmap crops to MAIN-frame viewport CSS pixels, not physical screen pixels.
// Await each operation in a gesture. Overlapping calls fail Busy instead of
// silently reordering down/up; errors and context cancellation are observable.
// Cancellation after native submission does not undo an already sent event.
//
// Cancel is a best-effort cleanup of buttons held through this API in the SAME
// document: move outside, release each held button with clickCount=0, move out
// again. It is NOT DOM pointercancel, blur, touch cancellation or OS SendInput;
// pointer-capture handlers may still receive release/click. A product gesture
// gate should disarm before cleanup if committing a click would be unwanted.
// Navigating discards old held state; it never sends old releases into a new
// document. Ordinary page features, native bindings and permissions are intact.
func (p *Page) MouseInput(ctx context.Context, e BrowserMouseEvent) error {
	if ctx == nil {
		return errors.New("mygo: mouse input context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := platform.ValidateBrowserMouseEvent(e); err != nil {
		return err
	}
	if p == nil || p.w == nil {
		return errDestroyed
	}
	// pageContext uses Window.mu and is safe before UI dispatch. Pin the
	// caller's current document, including a navigation while the call queues.
	expectedDocument := p.w.pageContext()
	done := make(chan error, 1)
	var once sync.Once
	finish := func(err error) { once.Do(func() { deliver(done, err) }) }
	stop := context.AfterFunc(ctx, func() { finish(ctx.Err()) })
	defer stop()
	ran := false
	onMain(func() {
		ran = true
		if err := ctx.Err(); err != nil {
			finish(err)
			return
		}
		if p == nil || p.w == nil || p.w.native == nil {
			finish(errDestroyed)
			return
		}
		if p.w.content != nil {
			finish(errNoPage)
			return
		}
		native := platform.BrowserMouseInputProvider(p.w.native)
		if native == nil {
			finish(ErrBrowserMouseInputUnsupported)
			return
		}
		doc := p.w.pageContext()
		if doc == nil || doc != expectedDocument || doc.Err() != nil {
			finish(context.Canceled)
			return
		}
		state := browserMouseStates[p.w]
		if state == nil {
			w := p.w
			w.OnClosed(func() { delete(browserMouseStates, w) })
		}
		if state == nil || state.document != doc {
			state = &browserMouseState{document: doc}
			browserMouseStates[p.w] = state
		}
		if state.busy {
			finish(ErrBrowserMouseInputBusy)
			return
		}
		state.busy = true
		stopDocument := context.AfterFunc(doc, func() { finish(doc.Err()) })
		events := []BrowserMouseEvent{e}
		if e.Kind == BrowserMouseCancel {
			events = platform.BrowserMouseCancelEvents(state.held)
		}
		end := func(err error) { state.busy = false; stopDocument(); finish(err) }
		var send func(int)
		send = func(index int) {
			if err := doc.Err(); err != nil {
				end(err)
				return
			}
			if err := ctx.Err(); err != nil {
				end(err)
				return
			}
			if p.w.native == nil {
				end(errDestroyed)
				return
			}
			n := platform.BrowserMouseInputProvider(p.w.native)
			if n == nil {
				end(ErrBrowserMouseInputUnsupported)
				return
			}
			if index == len(events) {
				end(nil)
				return
			}
			input := events[index]
			// Conservative after submission, including ambiguous native failures:
			// a later Cancel still tries to release a possibly dispatched press.
			state.held |= input.Buttons
			n.DispatchBrowserMouseInput(input, func(err error) {
				if err != nil {
					end(err)
					return
				}
				state.held = input.Buttons
				send(index + 1)
			})
		}
		send(0)
	})
	if !ran {
		return errDestroyed
	}
	err := await(done)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
