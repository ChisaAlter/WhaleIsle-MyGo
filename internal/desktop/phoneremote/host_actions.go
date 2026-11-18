package phoneremote

import (
	"fmt"
	"math"
	"strings"
)

// HostActions are the desktop surfaces the remote shell's host routes call:
// openSettings (also used by openGallery with the appearance section) and
// saveConfig. The Go app supplies them; a nil callback keeps the native
// "not opened" answer and an explicit unavailable reason instead of faking
// success.
type HostActions struct {
	OpenSettings func(section string) bool
	SaveConfig   func(patch map[string]any) (map[string]any, error)
}

const maxHostActionInFlight = 8

// SetHostActions installs the desktop callbacks. They run asynchronously from a
// bounded goroutine, so they must not block on the runtime lock.
func (r *Runtime) SetHostActions(actions HostActions) {
	r.mu.Lock()
	r.hostActions = actions
	r.mu.Unlock()
}

func validHostActionID(id string) bool {
	return id != "" && len(id) <= 20 && strings.Trim(id, "0123456789") == ""
}

func validateSaveConfigPatch(patch map[string]any) (map[string]any, error) {
	sanitized := make(map[string]any, len(patch))
	for key, value := range patch {
		switch key {
		case "remoteEnabled", "remoteLanTls":
			b, ok := value.(bool)
			if !ok {
				return nil, fmt.Errorf("%s must be a boolean", key)
			}
			sanitized[key] = b
		case "remotePort":
			var port float64
			switch n := value.(type) {
			case float64:
				port = n
			case int:
				port = float64(n)
			case int64:
				port = float64(n)
			default:
				return nil, fmt.Errorf("remotePort must be an integer from 1024 to 65535")
			}
			if math.IsNaN(port) || math.IsInf(port, 0) || math.Trunc(port) != port || port < 1024 || port > 65535 {
				return nil, fmt.Errorf("remotePort must be an integer from 1024 to 65535")
			}
			sanitized[key] = int(port)
		case "remoteMode", "remoteBindAddress", "remoteAppBaseUrl", "locale", "theme":
			s, ok := value.(string)
			if !ok || len(s) > 512 || strings.ContainsRune(s, '\x00') {
				return nil, fmt.Errorf("%s must be a string", key)
			}
			sanitized[key] = s
		default:
			return nil, fmt.Errorf("unsupported config field: %s", key)
		}
	}
	return sanitized, nil
}

func (r *Runtime) dispatchHostAction(child Child, kind, id, section string, patch map[string]any) {
	if !validHostActionID(id) {
		return
	}
	r.mu.Lock()
	active := r.child == child && !r.closing
	r.mu.Unlock()
	if !active || child == nil {
		return
	}
	if r.hostActionInFlight.Add(1) > maxHostActionInFlight {
		r.hostActionInFlight.Add(-1)
		if kind == "host-open-settings" {
			_ = r.sendControl(child, map[string]any{"action": "host-open-settings-result", "id": id, "opened": false})
		} else if kind == "host-save-config" {
			_ = r.sendControl(child, map[string]any{"action": "host-save-config-result", "id": id, "ok": false, "error": "host action busy"})
		}
		return
	}
	go func() {
		defer r.hostActionInFlight.Add(-1)
		r.handleHostAction(child, kind, id, section, patch)
	}()
}

// handleHostAction admits only the active bridge's bounded request. Revoked or
// replaced children cannot open desktop UI or persist a stale configuration.
func (r *Runtime) handleHostAction(child Child, kind, id, section string, patch map[string]any) {
	if child == nil || !validHostActionID(id) {
		return
	}
	r.mu.Lock()
	if r.child != child || r.closing {
		r.mu.Unlock()
		return
	}
	actions := r.hostActions
	r.mu.Unlock()
	if kind == "host-open-settings" {
		opened := false
		defer func() {
			if recover() != nil {
				opened = false
			}
			_ = r.sendControl(child, map[string]any{"action": "host-open-settings-result", "id": id, "opened": opened})
		}()
		if actions.OpenSettings != nil && len(section) <= 200 && !strings.ContainsRune(section, '\x00') {
			opened = actions.OpenSettings(section)
		}
		return
	}
	if kind != "host-save-config" {
		return
	}
	result := map[string]any{"action": "host-save-config-result", "id": id, "ok": false}
	defer func() {
		if recover() != nil {
			result = map[string]any{"action": "host-save-config-result", "id": id, "ok": false, "error": "host action failed"}
		}
		_ = r.sendControl(child, result)
	}()
	sanitized, err := validateSaveConfigPatch(patch)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	if actions.SaveConfig == nil {
		result["error"] = "saveConfig requires the desktop host"
	} else if saved, err := actions.SaveConfig(sanitized); err != nil {
		result["error"] = err.Error()
	} else {
		result["ok"], result["config"] = true, saved
	}
}
