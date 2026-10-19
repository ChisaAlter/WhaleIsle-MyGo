package settings

func (s *Store) UploadNotice() (uint64, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.uploadNoticeSequence, s.uploadNotice
}
func (s *Store) DismissUploadNotice(sequence uint64) {
	s.mu.Lock()
	if s.uploadNoticeSequence == sequence {
		s.uploadNotice = ""
	}
	s.mu.Unlock()
}
