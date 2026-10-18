package terminal

// SetReadOnly blocks terminal input without disabling local selection, copy or
// scrollback. Entering read-only mode discards queued input; restoring control
// never replays it. A write already handed to Conn cannot be recalled, so remote
// connections must also enforce their own current attachment/control identity.
func (t *Terminal) SetReadOnly(readOnly bool) {
	if t == nil {
		return
	}
	t.in.mu.Lock()
	t.in.readOnly = readOnly
	if readOnly {
		t.in.queue = nil
	}
	t.in.mu.Unlock()
}

// ReadOnly reports whether input is blocked. It is safe from any goroutine.
func (t *Terminal) ReadOnly() bool {
	if t == nil {
		return true
	}
	t.in.mu.Lock()
	defer t.in.mu.Unlock()
	return t.in.readOnly
}

func (t *Terminal) canCopy() bool {
	return t != nil && (t.opts.CanCopy == nil || t.opts.CanCopy())
}
