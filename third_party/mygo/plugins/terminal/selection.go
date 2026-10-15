package terminal

// SelectionText returns only the current user-selected text. It does not read
// the full scrollback or copy anything to the clipboard.
func (t *Terminal) SelectionText() (string, bool) {
	if t == nil {
		return "", false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.term == nil {
		return "", false
	}
	return t.term.SelectionText()
}
