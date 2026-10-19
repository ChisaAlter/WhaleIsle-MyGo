package schedule

// DeletionNotice belongs to one settled mutation on one Host attachment. It
// outlives the initiating detail, unlike its draft and loading state.
type DeletionNotice struct {
	ConnectionEpoch uint64
	Sequence        uint64
	Outcome         string // deleted, alreadyAbsent, failed
}

func (s *Store) publishDeletionLocked(outcome string) {
	s.deleteSequence++
	s.deleteNotice = &DeletionNotice{ConnectionEpoch: s.connectionEpoch, Sequence: s.deleteSequence, Outcome: outcome}
	s.deletePublished = false
}

// ConsumeDeletionNotice returns the current result and whether it has yet to
// be presented. The retained identity lets the app close it after Host detach
// without replaying it on every paint or depending on a still-open task panel.
func (s *Store) ConsumeDeletionNotice() (notice DeletionNotice, fresh bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.deleteNotice == nil || s.deleteNotice.ConnectionEpoch != s.connectionEpoch {
		return DeletionNotice{}, false
	}
	notice, fresh = *s.deleteNotice, !s.deletePublished
	s.deletePublished = true
	return notice, fresh
}

func (s *Store) DismissDeletionNotice(notice DeletionNotice) {
	s.mu.Lock()
	if s.deleteNotice != nil && s.deleteNotice.Sequence == notice.Sequence && s.deleteNotice.ConnectionEpoch == notice.ConnectionEpoch {
		s.deleteNotice = nil
	}
	s.mu.Unlock()
}
