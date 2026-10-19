package main

// flushInactiveGeneralFontEdits runs on the UI thread. It reads the cached
// editor without creating one and flushes only General's actual family edits.
// Call at mainView entry with false, or before close-state saving with true.
func (s *mainState) flushInactiveGeneralFontEdits(force bool) bool {
	s.mu.Lock()
	editor := s.appearanceEditor
	generalActive := s.settingsOpen && s.settingsDesktopActive
	s.mu.Unlock()
	if editor == nil || generalActive && !force {
		return true
	}
	return editor.FlushGeneralFontEdits(force)
}
