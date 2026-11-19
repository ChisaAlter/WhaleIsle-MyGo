//go:build !release

package text

// UseFixedMetrics lays text out with the engine that needs no fonts: every rune
// takes half an em and nothing is drawn. A test that measures a layout pins it
// there, so the numbers describe the composition rather than whichever faces the
// machine happens to resolve. Release builds carry no such switch: the file is
// behind !release, as the tester that calls it is.
func (s *System) UseFixedMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eng = &stubEngine{}
	clear(s.fonts)
	s.clearLayouts()
	clear(s.marks)
	s.gen++
}
