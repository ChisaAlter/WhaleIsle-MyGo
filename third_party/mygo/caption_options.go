package mygo

// WindowsCaptionButtonOptions gives a hidden title bar inset, rounded window
// controls on Windows. Sizes and offsets are DIPs, independently of
// TitleBarHeight: the controls overlay the content instead of adding a row.
// A nil WindowsCaptionButtons keeps the system-style controls. Other platforms
// ignore this option. GlyphSize is the square viewport of rounded vector glyphs.
type WindowsCaptionButtonOptions struct {
	Width, Height           int
	Top, Right              int
	CornerRadius, GlyphSize int
}
