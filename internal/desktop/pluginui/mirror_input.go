package pluginui

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type mirrorPointerEvent struct {
	event         ui.InputEvent
	width, height float32
}
type mirrorGesture struct {
	frame    *MirrorFrame
	row      MirrorOccurrence
	down     mirrorPointerEvent
	release  chan mirrorPointerEvent // one bounded release, never dropped while down awaits native completion
	ctx      context.Context
	cancel   context.CancelFunc
	released bool
	portal   string
}

func mirrorTypes(kind mygo.BrowserMouseKind) []string {
	switch kind {
	case mygo.BrowserMousePress:
		return []string{"pointerdown", "mousedown"}
	case mygo.BrowserMouseRelease:
		return []string{"pointerup", "mouseup", "click"}
	}
	return nil
}

// MirrorPointer pins one native down/up gesture to the original captured frame.
// A fast up queues while down is in flight; no new capture can replace that frame.
func (o *Owner) MirrorPointer(f *MirrorFrame, row MirrorOccurrence, event ui.InputEvent, width, height float32) bool {
	return o.mirrorPointer(f, row, event, width, height, "")
}
func (o *Owner) mirrorPointer(f *MirrorFrame, row MirrorOccurrence, event ui.InputEvent, width, height float32, portal string) bool {
	if event.Kind != ui.InputPointerDown && event.Kind != ui.InputPointerUp && event.Kind != ui.InputBlur {
		return false
	}
	o.mu.Lock()
	m := o.mirror
	if m == nil {
		o.mu.Unlock()
		return true
	}
	if event.Kind == ui.InputBlur {
		if m.gesture != nil {
			m.gesture.cancel()
		}
		o.mu.Unlock()
		return true
	}
	if event.Kind == ui.InputPointerUp {
		g := m.gesture
		if g != nil && !g.released {
			g.released = true
			if event.Button != 0 || row.Handle != g.row.Handle || portal != g.portal || width != g.down.width || height != g.down.height || event.X < 0 || event.Y < 0 || event.X > width || event.Y > height {
				g.cancel()
			} else {
				g.release <- mirrorPointerEvent{event, width, height}
			}
		}
		o.mu.Unlock()
		return true
	}
	valid := event.Button == 0 && width > 0 && height > 0 && event.X >= 0 && event.Y >= 0 && event.X <= width && event.Y <= height && o.mirrorCurrentLocked(f) && m.gesture == nil && !m.inputBusy
	if !valid {
		m.pointerStage = fmt.Sprintf("down-rejected: button=%d local=(%.2f,%.2f)/%.2fx%.2f current=%t gesture=%t busy=%t framegen=%d currentgen=%d", event.Button, event.X, event.Y, width, height, o.mirrorCurrentLocked(f), m.gesture != nil, m.inputBusy, func() uint64 {
			if f == nil {
				return 0
			}
			return f.generation
		}(), m.generation)
		o.mu.Unlock()
		return true
	}
	x, y := row.Rect.X+float64(event.X/width)*row.Rect.W, row.Rect.Y+float64(event.Y/height)*row.Rect.H
	visible := row.VisibleRect
	if !mirrorFragmentValid(visible) || x < visible.X || y < visible.Y || x > visible.X+visible.W || y > visible.Y+visible.H {
		o.mu.Unlock()
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	g := &mirrorGesture{portal: portal, frame: f, row: row, down: mirrorPointerEvent{event, width, height}, release: make(chan mirrorPointerEvent, 1), ctx: ctx, cancel: cancel}
	m.gesture = g
	m.inputBusy = true
	o.mu.Unlock()
	go o.runMirrorGesture(m, g)
	return true
}

func (o *Owner) runMirrorGesture(m *mirrorState, g *mirrorGesture) {
	o.mirrorPointerStage(m, "await-source-capture")
	err := o.awaitMirrorCapture(g.ctx, m)
	if err == nil {
		o.mirrorPointerStage(m, "down-dispatch")
		err = o.dispatchMirror(g, g.down)
	}
	o.mirrorPointerStage(m, "down-complete:"+fmt.Sprint(err))
	if err == nil {
		select {
		case up := <-g.release:
			err = o.dispatchMirror(g, up)
			o.mirrorPointerStage(m, "up-complete:"+fmt.Sprint(err))
			if err == nil {
				focus, e := mirrorEval[*mirrorTextFocus](g.ctx, g.frame.source.Page(), "window.__whaleInlineMirror.textFocus()")
				o.mu.Lock()
				m.textFocusError = fmt.Sprint(e)
				o.mu.Unlock()
				if e == nil {
					o.bindMirrorOriginalFocus(g.frame, focus)
				}
			}
		case <-g.ctx.Done():
			err = g.ctx.Err()
		}
	}
	// Cleanup is always confined to the same document and suppresses all original
	// mouse handlers before native Cancel can release a possibly held button.
	if err != nil {
		o.cancelMirrorGesture(g)
	}
	g.cancel()
	o.mu.Lock()
	if m.gesture == g {
		m.gesture = nil
		m.inputBusy = false
		m.error = ""
		if err != nil {
			m.error = err.Error()
		}
		o.invalidateMirrorLocked()
	}
	o.mu.Unlock()
	o.notify()
}

func (o *Owner) dispatchMirror(g *mirrorGesture, input mirrorPointerEvent) error {
	f, row, event := g.frame, g.row, input.event
	if g.ctx.Err() != nil {
		return g.ctx.Err()
	}
	if !o.mirrorCurrent(f) {
		return errSessionUnavailable
	}
	now, err := mirrorEval[mirrorGeometry](g.ctx, f.source.Page(), "window.__whaleInlineMirror.snapshot()")
	if err != nil {
		return fmt.Errorf("pointer geometry query: %w", err)
	}
	if !sameMirrorInputGeometry(now, f.geometry) {
		return errSessionUnavailable
	}
	x, y := row.Rect.X+float64(event.X/input.width)*row.Rect.W, row.Rect.Y+float64(event.Y/input.height)*row.Rect.H
	kind := mygo.BrowserMousePress
	buttons := uint8(1)
	if event.Kind == ui.InputPointerUp {
		kind = mygo.BrowserMouseRelease
		buttons = 0
	}
	nonce := rand.Text()
	arm := map[string]any{"documentKey": now.DocumentKey, "revision": now.Revision, "subject": f.subject, "handle": row.Handle, "entry": row.Entry, "turn": row.Turn, "seq": row.Seq, "nonce": nonce, "x": x, "y": y, "types": mirrorTypes(kind), "portal": g.portal}
	raw, _ := json.Marshal(arm)
	accepted, err := mirrorEval[struct {
		OK bool `json:"ok"`
	}](g.ctx, f.source.Page(), "window.__whaleInlineMirror.arm("+string(raw)+")")
	if err != nil {
		return fmt.Errorf("pointer arm query: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = f.source.Page().EvalContext(ctx, "window.__whaleInlineMirror.disarm("+fmt.Sprintf("%q", nonce)+")")
	}()
	if !accepted.OK || !o.mirrorCurrent(f) {
		return errSessionUnavailable
	}
	err = f.source.Page().MouseInput(g.ctx, mygo.BrowserMouseEvent{Kind: kind, X: x, Y: y, Button: mygo.BrowserMouseLeft, Buttons: buttons, ClickCount: 1})
	if err != nil {
		return fmt.Errorf("pointer native operation: %w", err)
	}
	receipt, err := mirrorEval[struct {
		OK bool `json:"ok"`
	}](g.ctx, f.source.Page(), "window.__whaleInlineMirror.disarm("+fmt.Sprintf("%q", nonce)+")")
	if err != nil {
		return fmt.Errorf("pointer receipt query: %w", err)
	}
	if !receipt.OK {
		return errors.New("plugin mirror: original target refused input")
	}
	return nil
}

func (o *Owner) cancelMirrorGesture(g *mirrorGesture) {
	f := g.frame
	if f == nil || f.source == nil || f.source.IsDestroyed() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	blocked, err := mirrorEval[bool](ctx, f.source.Page(), "window.__whaleInlineMirror?.cancel("+fmt.Sprintf("%q", f.geometry.DocumentKey)+")")
	if err != nil || !blocked {
		return
	}
	_ = f.source.Page().MouseInput(ctx, mygo.BrowserMouseEvent{Kind: mygo.BrowserMouseCancel})
	_, _ = f.source.Page().EvalContext(ctx, "window.__whaleInlineMirror?.resume("+fmt.Sprintf("%q", f.geometry.DocumentKey)+")")
}
