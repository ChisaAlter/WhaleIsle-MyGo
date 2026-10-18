package browserpreview

import (
	"context"
	"errors"
	"runtime"
)

func (b *Controller) openBrowserDevTools(t *Tab) {
	if t == nil || b.current() != t || t.Host == nil || b.parent == nil || b.parent.IsDestroyed() {
		return
	}
	h, epoch := t.Host, t.Host.BrowserPageVersion()
	if !b.browserToolsCurrent(t, h, epoch) {
		return
	}
	err := h.OpenBrowserDevTools(epoch) // Synchronous UI-thread dispatch, no queued old-tab callback.
	if !b.browserToolsCurrent(t, h, epoch) {
		return
	}
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Error = err.Error()
		}
	} else {
		t.Error = ""
		_, _, t.ActionMessage = browserDevToolsCopy(runtime.GOOS)
	}
	b.update()
}
