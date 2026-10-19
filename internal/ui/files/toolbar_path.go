package files

import (
	"github.com/egoist/mygo/ui"
	"strings"
)

// Path presentation and copy do not depend on a local filesystem mount. They
// use the selected workspace's lexical Host path and grant no read authority.
func toolbarAbsolutePath(m Model, path string) string {
	if values, ok := treePathCopies(m, path); ok {
		return values.absolute
	}
	return path
}
func revealToolbarPath(m Model, path, full string) {
	if m.OnReveal == nil {
		return
	}
	withToolbarPath(m, path, full, m.OnReveal)
}

func withToolbarPath(m Model, path, full string, action func(string)) {
	if action == nil {
		return
	}
	values, ok := treePathCopies(m, path)
	if !ok {
		return
	}
	if m.ReadRemoteContext != nil || m.ListRemoteContext != nil {
		action(values.absolute)
		return
	}
	if full != "" {
		withPreviewPath(m, path, full, action)
	}
}
func paintToolbarPath(c *ui.Context, m Model, path string, muted ui.Color) {
	full := toolbarAbsolutePath(m, path)
	split := strings.LastIndexAny(full, "/\\")
	directory, name := "", full
	if split >= 0 {
		directory, name = full[:split+1], full[split+1:]
	}
	ui.Row(c).Grow(1).MinWidth(0).Gap(0).Tooltip(previewPathTooltip(m, path)).Children(func() {
		if directory != "" {
			ui.Text(c, directory).FontSize(11).TextColor(muted).SingleLine().Grow(1).MinWidth(0)
		}
		ui.Text(c, name).FontSize(11).Bold().SingleLine().Shrink(1).MinWidth(0).Tooltip(full)
	})
}

func previewPathTooltip(m Model, path string) string {
	text := toolbarAbsolutePath(m, path)
	if m.StatNote != nil && *m.StatNote != "" {
		text += "\n" + *m.StatNote
	}
	return text
}
