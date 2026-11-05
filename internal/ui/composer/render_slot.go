package composer

import "github.com/egoist/mygo/ui"

// A contribution owns only its semantic seat. The editor, native draft owner,
// IME, send transport and adjacent seats remain independently native.
func renderComposerSlot(c *ui.Context, m Model, name string, native func()) {
	if m.RenderSlot != nil {
		m.RenderSlot(c, name, native)
	} else {
		native()
	}
}
