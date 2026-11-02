package ui

// ScrollbarsVisible overrides automatic scrollbar visibility for this scroll
// container alone. False hides the bars, including on hover, and prevents
// presses on their hidden tracks. True shows them only on overflowing axes.
// Scrolling by wheel, touchpad and keyboard is unchanged. Without this method,
// the theme's automatic visibility and hover behavior apply.
func (e *node) ScrollbarsVisible(visible bool) *node {
	e.barVisible = [2]bool{true, visible}
	return e
}
