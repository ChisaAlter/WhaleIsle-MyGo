package phoneremote

// HostActions are the desktop surfaces the remote shell's host routes call:
// openSettings (also used by openGallery with the appearance section) and
// saveConfig. The Go app supplies them; a nil callback keeps the native
// "not opened" answer and an explicit unavailable reason instead of faking
// success.
type HostActions struct {
	OpenSettings func(section string) bool
	SaveConfig   func(patch map[string]any) (map[string]any, error)
}

// SetHostActions installs the desktop callbacks. They run on the bridge reader
// goroutine, so they must not block on the runtime lock.
func (r *Runtime) SetHostActions(actions HostActions) {
	r.mu.Lock()
	r.hostActions = actions
	r.mu.Unlock()
}
