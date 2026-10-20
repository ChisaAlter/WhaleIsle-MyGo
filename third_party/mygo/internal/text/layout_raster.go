package text

// WithRasterSystem borrows this System's native engine for one synchronous
// paint into private glyph atlases. It preserves registered fonts, fallback
// and native rendering settings without modifying the window's shared atlas.
// The callback must use only the supplied System, must not retain it or start
// goroutines using it, and must not call methods of the original System.
// A scene may retain the private atlases for rendering after the callback:
// they are no longer written once the callback returns.
func (s *System) WithRasterSystem(paint func(*System)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	private := newSystem()
	private.eng = s.engine()
	paint(private)
}
