package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// SpinnerOptions configure a Spinner. Zero size is 16; zero color is the
// foreground.
type SpinnerOptions struct {
	Size  float32
	Color ui.Color
}

// Spinner is coss's Spinner: lucide loader-circle turning once a second
// (animate-spin), role status, named "Loading".
func Spinner(c *ui.Context, o SpinnerOptions) ui.Element {
	t := tok(c)
	size := o.Size
	if size == 0 {
		size = design.IconSM
	}
	col := o.Color
	if col.A == 0 {
		col = t.Foreground
	}
	e := ui.Icon(c, IconLoader.SVG()).Size(size, size).Shrink(0).TextColor(col).Role(ui.RoleStatus).Label("Loading")
	if forced == 0 && !c.Preferences().ReduceMotion {
		e.Rotate(e.Loop("coss.spin", design.DurationSpin, design.EaseLinear) * 360)
	}
	return e
}
