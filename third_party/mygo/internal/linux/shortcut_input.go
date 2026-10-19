//go:build linux && (amd64 || arm64)

package linux

func (w *window) ShortcutInputEnabled() bool {
	return !w.closed && w.win != 0 && gtkWidgetIsSensitive(w.win)
}
