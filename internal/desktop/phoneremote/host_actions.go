package phoneremote

import "strings"

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

// handleHostAction admits only the active bridge's bounded request. Revoked or
// replaced children cannot open desktop UI or persist a stale configuration.
func (r *Runtime) handleHostAction(child Child, kind, id, section string, patch map[string]any) {
	if child == nil || id == "" || len(id) > 20 || strings.Trim(id, "0123456789") != "" {
		return
	}
	r.mu.Lock()
	if r.child != child {
		r.mu.Unlock()
		return
	}
	actions := r.hostActions
	r.mu.Unlock()
	if kind == "host-open-settings" {
		opened := false
		if actions.OpenSettings != nil && len(section) <= 200 && !strings.ContainsRune(section, '\x00') {
			opened = actions.OpenSettings(section)
		}
		_ = r.sendControl(child, map[string]any{"action": "host-open-settings-result", "id": id, "opened": opened})
		return
	}
	if kind != "host-save-config" {
		return
	}
	result := map[string]any{"action": "host-save-config-result", "id": id, "ok": false}
	if actions.SaveConfig == nil {
		result["error"] = "saveConfig requires the desktop host"
	} else if saved, err := actions.SaveConfig(patch); err != nil {
		result["error"] = err.Error()
	} else {
		result["ok"], result["config"] = true, saved
	}
	_ = r.sendControl(child, result)
}
