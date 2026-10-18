package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets/diff"
)

// RecordedPatchView renders a Host snapshot without repository reads or actions.
func RecordedPatchView(c *ui.Context, source string, split, wrap bool, st diff.Style) {
	w := &Workbench{}
	w.View.Split, w.View.Wrap = split, wrap
	w.View.update(source)
	w.View.setAll(true)
	paintPatchFiles(c, Model{Workbench: w}, source, st)
}
