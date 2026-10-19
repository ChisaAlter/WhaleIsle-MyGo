//go:build windows && (amd64 || arm64)

package windows

var procShortcutIsWindowEnabled = user32.NewProc("IsWindowEnabled")

func (w *window) ShortcutInputEnabled() bool {
	if w.closed || w.hwnd == 0 {
		return false
	}
	enabled, _, _ := procShortcutIsWindowEnabled.Call(w.hwnd)
	return enabled != 0
}
