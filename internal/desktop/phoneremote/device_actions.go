package phoneremote

import (
	_ "embed"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
)

//go:embed device-actions.cjs
var deviceActionsScript string

// Device contains only public metadata from DshdRemote.snapshot, never secrets.
type Device struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"createdAt"`
	LastSeenAt string `json:"lastSeenAt"`
}

type DeviceActionState struct {
	ID, Operation, DeviceID string
	Busy, Applied           bool
	Error, Message          string
}

// DeviceAction queues one original-runtime operation. It never starts a daemon
// or accepts a display label as an identity. Completion is observed in Snapshot.
func (r *Runtime) DeviceAction(operation, id, name string) error {
	if operation != "rename" && operation != "unbind" && operation != "rotate" {
		return errors.New("unsupported phone device action")
	}
	name = strings.TrimSpace(name)
	if operation == "rename" && (name == "" || len(utf16.Encode([]rune(name))) > 120 || strings.IndexFunc(name, unicode.IsControl) >= 0) {
		return errors.New("device name must contain 1–120 characters without control characters")
	}
	r.mu.Lock()
	if r.deviceAction.Busy {
		r.mu.Unlock()
		return errors.New("a phone device action is already running")
	}
	child := r.child
	if child == nil || !r.state.Listening || !r.state.DevicesLoaded {
		r.mu.Unlock()
		return errors.New("refresh the phone runtime before managing devices")
	}
	if operation != "rotate" {
		found := false
		for _, device := range r.state.Devices {
			if device.ID == id {
				found = true
				break
			}
		}
		if !found || id == "" || len(id) > 1024 {
			r.mu.Unlock()
			return errors.New("the paired device is no longer available")
		}
	}
	r.deviceSequence++
	requestID := strconv.FormatUint(r.deviceSequence, 10)
	done := make(chan struct{})
	r.deviceActionDone = done
	r.deviceAction = DeviceActionState{ID: requestID, Operation: operation, DeviceID: id, Busy: true}
	if operation != "rename" {
		r.state.PairingURL = ""
		r.state.PairingImage = nil
		r.state.ExpiresAt = time.Time{}
	}
	r.mu.Unlock()
	go func() {
		r.op.Lock()
		r.mu.Lock()
		current := r.child == child && r.deviceAction.ID == requestID && r.deviceAction.Busy
		r.mu.Unlock()
		var err error
		if current {
			err = r.sendControl(child, map[string]any{"action": "device-action", "id": requestID, "operation": operation, "deviceId": id, "name": name})
		}
		r.op.Unlock()
		if !current {
			return
		}
		if err == nil {
			select {
			case <-done:
				return
			case <-time.After(45 * time.Second):
			}
		}
		r.op.Lock()
		defer r.op.Unlock()
		r.mu.Lock()
		current = r.child == child && r.deviceAction.ID == requestID && r.deviceAction.Busy
		if current {
			code := "timeout"
			if err != nil {
				code = "control"
			}
			r.finishDeviceActionLocked(code, false)
		}
		r.mu.Unlock()
		if current {
			// Unknown completion must not leave an old offer or connection usable
			// in the UI. A manual refresh can reconcile the original store.
			r.stopLocked()
		}
	}()
	return nil
}

// Called with r.mu held. Only fixed messages are exposed; runtime exception
// text can contain paths, pairing URLs, or credentials.
func (r *Runtime) finishDeviceActionLocked(code string, applied bool) {
	if !r.deviceAction.Busy {
		return
	}
	r.deviceAction.Busy = false
	r.deviceAction.Applied = applied
	if code != "" {
		switch code {
		case "device-missing":
			r.deviceAction.Error = "The paired device is no longer available. Refresh and try again."
		case "invalid-name":
			r.deviceAction.Error = "Enter a device name with 1–120 characters."
		case "timeout":
			r.deviceAction.Error = "The operation timed out. Its result is unknown; refresh before retrying."
		case "stopped":
			r.deviceAction.Error = "The remote runtime stopped. Refresh to check whether the operation was applied."
		default:
			r.deviceAction.Error = "The phone device operation failed. Refresh the runtime and try again."
		}
		if applied {
			r.deviceAction.Error = "The device was unbound, but the remote runtime could not restart. Refresh to reconnect other devices."
		}
	} else {
		switch r.deviceAction.Operation {
		case "rename":
			r.deviceAction.Message = "Device renamed."
		case "unbind":
			r.deviceAction.Message = "Device unbound. Other paired devices can reconnect when the runtime is ready."
		case "rotate":
			r.deviceAction.Message = "Pairing offer renewed. Existing device identities are unchanged; older offers retain their original expiry."
		}
	}
	if r.deviceActionDone != nil {
		close(r.deviceActionDone)
		r.deviceActionDone = nil
	}
}

// WaitDeviceAction waits for the currently queued operation, including stop or
// timeout. It is intended for a background UI invalidation goroutine.
func (r *Runtime) WaitDeviceAction() {
	r.mu.Lock()
	done := r.deviceActionDone
	r.mu.Unlock()
	if done != nil {
		<-done
	}
}
