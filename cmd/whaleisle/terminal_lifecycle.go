package main

// flushTerminalResizeLocked preserves the last visible grid before a drawer
// hide or workspace switch. The caller already holds mainState.mu, like the
// terminal session lifecycle. It never creates, restarts or closes a shell.
func (s *mainState) flushTerminalResizeLocked() {
	if s.terminalPanel != nil {
		s.terminalPanel.FlushResize()
	}
}
