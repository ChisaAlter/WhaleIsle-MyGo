//go:build darwin

package darwin

func (w *window) ShortcutInputEnabled() bool {
	return !w.closed && w.win != 0 && send(w.win, "attachedSheet") == 0 && send(w.b.app, "modalWindow") == 0
}
