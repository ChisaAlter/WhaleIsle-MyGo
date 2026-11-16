package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets/diff"
)

// RecordedPatchView renders a Host snapshot without repository reads or actions.
func RecordedPatchView(c *ui.Context, source string, split, wrap bool, st diff.Style) {
	paintRecordedPatchFiles(c, source, split, wrap, st)
}
