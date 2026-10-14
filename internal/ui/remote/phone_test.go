package remote

import (
	"reflect"
	"testing"

	"whaleisle/internal/desktop/phoneremote"
)

func TestPhoneSnapshotConnectURL(t *testing.T) {
	m := Model{
		RemoteEnabled:     true,
		RemoteMode:        "lan",
		RemotePort:        6767,
		RemoteBindAddress: "10.0.0.5",
		RemoteLanTls:      false,
		HasRemoteToken:    true,
		HarnessReady:      true,
		MuxConnected:      true,
	}
	s := phoneSnapshot(m)
	if s.Readiness != phoneremote.ReadinessReady {
		t.Fatalf("Readiness = %q missing=%v", s.Readiness, s.Missing)
	}
	if s.ConnectURL != "http://10.0.0.5:6767" {
		t.Fatalf("ConnectURL = %q", s.ConnectURL)
	}
}

func TestPhoneSnapshotMuxIncomplete(t *testing.T) {
	m := Model{
		RemoteEnabled:     true,
		RemoteMode:        "lan",
		RemotePort:        6767,
		RemoteBindAddress: "10.0.0.5",
		HasRemoteToken:    true,
		HarnessReady:      true,
		MuxConnected:      false,
	}
	s := phoneSnapshot(m)
	if s.Readiness != phoneremote.ReadinessIncomplete {
		t.Fatalf("Readiness = %q", s.Readiness)
	}
}

func TestPhoneSnapshotDeviceNames(t *testing.T) {
	m := Model{
		RemoteDeviceCount: 2,
		RemoteDeviceNames: []string{"iPhone", "Pad"},
	}
	s := phoneSnapshot(m)
	if s.DeviceCount != 2 {
		t.Fatalf("DeviceCount = %d", s.DeviceCount)
	}
	if !reflect.DeepEqual(s.DeviceNames, []string{"iPhone", "Pad"}) {
		t.Fatalf("DeviceNames = %v", s.DeviceNames)
	}
}
