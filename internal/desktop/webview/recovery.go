package webview

// PageFailure carries only the failed page's recovery capability. A callback
// retained by a dialog cannot reload a later navigation or a replacement Host.
type PageFailure struct {
	Kind, Message string
	Current       func() bool
	Reload        func()
}

func (h *Host) reportFailure(kind, message string) {
	h.mu.Lock()
	fn, epoch, win := h.failure, h.navigationEpoch, h.win
	if fn == nil || h.failureEpoch == epoch+1 {
		h.mu.Unlock()
		return
	}
	h.failureEpoch = epoch + 1
	h.mu.Unlock()
	current := func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.win == win && win != nil && !win.IsDestroyed() && h.navigationEpoch == epoch
	}
	fn(PageFailure{Kind: kind, Message: message, Current: current, Reload: func() {
		if current() {
			h.Reload()
		}
	}})
}
