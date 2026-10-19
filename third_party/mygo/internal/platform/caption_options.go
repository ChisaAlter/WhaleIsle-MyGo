package platform

// WindowsCaptionButtonOptions is the app-scoped hidden-caption geometry.
// A nil option preserves the backend's system-style controls.
type WindowsCaptionButtonOptions struct {
	Width, Height           int
	Top, Right              int
	CornerRadius, GlyphSize int
}
