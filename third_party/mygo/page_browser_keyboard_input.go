package mygo

import (
	"context"
	"errors"
	"sync"

	"github.com/egoist/mygo/internal/platform"
)

type BrowserTextKind = platform.BrowserTextKind
type BrowserTextModifiers = platform.BrowserTextModifiers
type BrowserTextEvent = platform.BrowserTextEvent

const (
	BrowserTextKeyDown           = platform.BrowserTextKeyDown
	BrowserTextKeyUp             = platform.BrowserTextKeyUp
	BrowserTextInsertText        = platform.BrowserTextInsertText
	BrowserTextComposition       = platform.BrowserTextComposition
	BrowserTextCancelComposition = platform.BrowserTextCancelComposition
	BrowserTextAlt               = platform.BrowserTextAlt
	BrowserTextCtrl              = platform.BrowserTextCtrl
	BrowserTextMeta              = platform.BrowserTextMeta
	BrowserTextShift             = platform.BrowserTextShift
)

var ErrBrowserTextInputUnsupported = platform.ErrUnsupported
var ErrBrowserTextInputBusy = errors.New("mygo: await the previous text input before sending another")

type browserTextState struct {
	document        context.Context
	busy, composing bool
}

// UI-thread confined. Closure removes the strong key; navigation discards only
// this API's composition ownership, never canceling into the next document.
var browserTextStates = map[*Window]*browserTextState{}

// BrowserTextInputAvailable reports an optional native input provider. Currently
// Windows WebView2 implements it. It opens no inspector and grants no scripting,
// clipboard or focus capability. Content windows have no browser input surface.
func (p *Page) BrowserTextInputAvailable() bool {
	return onMainValue(func() bool {
		return p != nil && p.w != nil && p.w.native != nil && p.w.content == nil && platform.BrowserTextInputProvider(p.w.native) != nil
	})
}

// TextInput sends one typed operation to the page's currently focused editable
// element. KeyDown/KeyUp carry no committed text; use InsertText for Unicode
// commits and Composition for uncommitted IME text. Selection/replacement offsets
// are UTF-16 units, independently of toolkit rune or byte indices. Missing native
// support returns ErrBrowserTextInputUnsupported (platform.ErrUnsupported).
//
// Await operations in order. Overlap fails Busy; context cancellation is
// observable but cannot undo an operation already submitted. Each invocation is
// pinned to its current document and backend. The caller must also guard its
// focus/session ownership across a gesture; this API never focuses an element,
// synthesizes HTML keyboard events or replays old key-up events after navigation.
// CancelComposition cancels only composition submitted by this API in this same
// document. It commits no text and does nothing if that ownership is absent.
func (p *Page) TextInput(ctx context.Context, e BrowserTextEvent) error {
	if ctx == nil {
		return errors.New("mygo: text input context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	e = platform.CloneBrowserTextEvent(e)
	if err := platform.ValidateBrowserTextEvent(e); err != nil {
		return err
	}
	if p == nil || p.w == nil {
		return errDestroyed
	}
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
		w := p.w
		if w.native == nil {
			finish(errDestroyed)
			return
		}
		if w.content != nil {
			finish(errNoPage)
			return
		}
		native := w.native
		provider := platform.BrowserTextInputProvider(native)
		if provider == nil {
			finish(ErrBrowserTextInputUnsupported)
			return
		}
		doc := w.pageContext()
		if doc == nil || doc != expectedDocument || doc.Err() != nil {
			finish(context.Canceled)
			return
		}
		state := browserTextStates[w]
		if state == nil {
			w.OnClosed(func() { delete(browserTextStates, w) })
		}
		if state == nil || state.document != doc {
			state = &browserTextState{document: doc}
			browserTextStates[w] = state
		}
		if state.busy {
			finish(ErrBrowserTextInputBusy)
			return
		}
		if e.Kind == BrowserTextCancelComposition && !state.composing {
			finish(nil)
			return
		}
		state.busy = true
		if e.Kind == BrowserTextComposition {
			state.composing = true
		}
		stopDocument := context.AfterFunc(doc, func() { finish(doc.Err()) })
		provider.DispatchBrowserTextInput(e, func(err error) {
			state.busy = false
			stopDocument()
			if w.native == nil {
				finish(errDestroyed)
				return
			}
			if w.native != native || w.pageContext() != doc || doc.Err() != nil {
				finish(context.Canceled)
				return
			}
			if ctx.Err() != nil {
				finish(ctx.Err())
				return
			}
			if err == nil && (e.Kind == BrowserTextCancelComposition || e.Kind == BrowserTextInsertText) {
				state.composing = false
			}
			finish(err)
		})
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
