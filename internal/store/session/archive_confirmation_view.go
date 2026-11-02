package session

// ArchiveConfirmation returns an independent copy of the current confirmation
// without projecting or cloning the conversation history. Nil means no Host
// refusal is waiting for the user, matching Snapshot().ArchiveConfirmation.
func (s *Store) ArchiveConfirmation() *ArchiveConfirmation {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.archive.waiting {
		return nil
	}
	return cloneArchiveConfirmation(&s.archive.confirmation)
}
