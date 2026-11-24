package interaction

// testView is the pre-publication Snapshot shape the package tests assert
// on: the published view plus the live question cards.
func (s *Store) testView() ViewState {
	v := s.Snapshot().Value
	v.QuestionCards = s.QuestionCards()
	return v
}
