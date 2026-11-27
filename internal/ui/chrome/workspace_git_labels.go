package chrome

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// captionGitStaticLabel paints a fixed-slot, single-line truncated label inside
// wide/cozy Git toolbar controls. Hovering or focusing never animates width or
// shifts adjacent titlebar controls.
func captionGitStaticLabel(c *ui.Context, text, part string, maxWidth float32) {
	size := theme.InterfaceSize(12)
	tx := ui.Text(c, text).Key("git-label-"+part).Font(c.Theme().Font).FontSize(size).FontWeight(400).
		FixedLineHeight(18).MinWidth(0).SingleLine().Padding(0, 0, 0, 4).TextColor(c.Theme().Text)
	if maxWidth > 0 {
		tx.MaxWidth(maxWidth)
	}
}
