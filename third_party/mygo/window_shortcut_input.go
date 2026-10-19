package mygo

import "github.com/egoist/mygo/internal/platform"

// ShortcutInputEnabled reports the native modal/sensitivity state for a
// revision-checked product shortcut. Unsupported backends do not grant it.
func (w *Window) ShortcutInputEnabled() bool {
	if w == nil {
		return false
	}
	return get(w, func(native platform.Window) bool {
		query, ok := native.(interface{ ShortcutInputEnabled() bool })
		return ok && query.ShortcutInputEnabled()
	})
}
