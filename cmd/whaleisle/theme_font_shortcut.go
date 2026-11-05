package main

// adjustFontSizes is shared by the fixed font shortcuts and the settings
// editor. The editor writes only changed Host typography fields and preserves
// any unfinished custom-theme draft.
func (s *mainState) adjustFontSizes(delta int) bool {
	editor := s.themeFontEditor(s.appearanceVisible)
	return editor != nil && editor.AdjustFontSizes(delta)
}
