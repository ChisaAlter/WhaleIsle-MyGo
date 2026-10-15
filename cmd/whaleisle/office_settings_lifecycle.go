package main

// Clear write-only tokens and pending reviews when native navigation or window
// lifecycle bypasses the Office panel's own close buttons. Hide can notify, so
// it must run after releasing the main state lock.
func (s *mainState) hideOfficeSettings() {
	s.mu.Lock()
	im := s.im
	s.mu.Unlock()
	if im != nil && im.Office.Snapshot().Open {
		im.Office.Hide()
	}
}
