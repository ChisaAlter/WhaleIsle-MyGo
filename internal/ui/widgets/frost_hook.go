package widgets

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// SurfaceDresser dresses a raised surface while a live backdrop is painted.
// chrome wires it at init time; widgets cannot import chrome without an import
// cycle. Nil keeps the plain themed background.
var SurfaceDresser func(e ui.Element, colors theme.Colors, dark bool, base ui.Color, radius float32) ui.Element

// FrostSurface applies the registered raised-surface dressing, or falls back
// to the plain background when no dresser is installed.
func FrostSurface(e ui.Element, colors theme.Colors, dark bool, base ui.Color, radius float32) ui.Element {
	if SurfaceDresser != nil {
		return SurfaceDresser(e, colors, dark, base, radius)
	}
	return e.Radius(radius).Background(base)
}
