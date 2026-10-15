package phoneremote

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type deviceControlChild struct{ writes chan []byte }
type deviceControlWriter struct{ writes chan []byte }

func (w deviceControlWriter) Write(p []byte) (int, error) {
	w.writes <- append([]byte(nil), p...)
	return len(p), nil
}
func (c *deviceControlChild) Input() io.Writer  { return deviceControlWriter{c.writes} }
func (c *deviceControlChild) Output() io.Reader { return strings.NewReader("") }
func (c *deviceControlChild) Wait() error       { return nil }
func (c *deviceControlChild) Kill()             {}

func TestDeviceActionControlAndGeneration(t *testing.T) {
	child := &deviceControlChild{writes: make(chan []byte, 1)}
	r := NewRuntime()
	r.child = child
	r.state = RuntimeState{Status: "ready", Listening: true, DevicesLoaded: true,
		Devices: []Device{{ID: "device-0001", Name: "Phone"}}}
	if err := r.DeviceAction("rename", "device-0001", " Renamed "); err != nil {
		t.Fatal(err)
	}
	var command map[string]string
	if err := json.Unmarshal(<-child.writes, &command); err != nil {
		t.Fatal(err)
	}
	if command["operation"] != "rename" || command["deviceId"] != "device-0001" || command["name"] != "Renamed" {
		t.Fatal("wrong device control command")
	}
	if err := r.DeviceAction("rotate", "", ""); err == nil {
		t.Fatal("concurrent mutation accepted")
	}
	result, _ := json.Marshal(map[string]any{"kind": "device-action-result", "id": command["id"], "operation": "rename", "applied": true})
	r.consume(&deviceControlChild{}, result)
	if !r.Snapshot().DeviceAction.Busy {
		t.Fatal("obsolete child completed current action")
	}
	r.consume(child, []byte(`{"kind":"device-action-result","id":"999","operation":"rename","applied":true}`))
	if !r.Snapshot().DeviceAction.Busy {
		t.Fatal("obsolete request completed current action")
	}
	r.consume(child, result)
	r.WaitDeviceAction()
	if got := r.Snapshot().DeviceAction; got.Busy || !got.Applied || got.Error != "" {
		t.Fatal("valid acknowledgement was not applied")
	}
	copy := r.Snapshot()
	copy.Devices[0].Name = "outside"
	if r.Snapshot().Devices[0].Name != "Phone" {
		t.Fatal("snapshot leaked mutable device slice")
	}
}

func TestDeviceActionValidationAndFailure(t *testing.T) {
	r := NewRuntime()
	if r.DeviceAction("rotate", "", "") == nil {
		t.Fatal("offline mutation accepted")
	}
	child := &deviceControlChild{writes: make(chan []byte, 1)}
	r.child = child
	r.state = RuntimeState{Status: "ready", Listening: true, RelayConnected: true, DevicesLoaded: true,
		PairingURL: "old-offer", Devices: []Device{{ID: "device-0001", Name: "Phone"}}}
	for _, name := range []string{" ", "line\nbreak", strings.Repeat("x", 121)} {
		if r.DeviceAction("rename", "device-0001", name) == nil {
			t.Fatal("invalid name accepted")
		}
	}
	if r.DeviceAction("unbind", "Phone", "") == nil {
		t.Fatal("display name accepted as identity")
	}
	if err := r.DeviceAction("unbind", "device-0001", ""); err != nil {
		t.Fatal(err)
	}
	var command map[string]string
	_ = json.Unmarshal(<-child.writes, &command)
	if r.Snapshot().PairingURL != "" || !r.Snapshot().DeviceAction.Busy {
		t.Fatal("old offer remained visible")
	}
	result, _ := json.Marshal(map[string]any{"kind": "device-action-result", "id": command["id"], "operation": "unbind", "code": "fictional-sensitive-error", "applied": true})
	r.consume(child, result)
	r.WaitDeviceAction()
	action := r.Snapshot().DeviceAction
	if action.Busy || !action.Applied || !strings.Contains(action.Error, "could not restart") || strings.Contains(action.Error, "fictional-sensitive") {
		t.Fatal("partial operation result or sanitized error lost")
	}
}
