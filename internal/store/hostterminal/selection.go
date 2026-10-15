package hostterminal

// CanCopySelection validates the displayed renderer's identity at the click.
// Reading a local selection does not require input control of the Host process.
func (s *Store) CanCopySelection(owner uint64, session, id string, a *Attachment) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.sessions[session]
	e := s.entries[terminalKey{session, id}]
	return a != nil && s.epoch == owner && s.visible == session && st != nil && st.active == id && e != nil && e.attachment == a && a.entry == e && a.parsed && s.currentLocked(a) && e.phase != "closed"
}
