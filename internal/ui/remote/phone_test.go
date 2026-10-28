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
		RemoteAppBaseURL:  "http://10.0.0.5:6767",
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

func TestPhoneSnapshotDeviceCountSources(t *testing.T) {
	for _, tt := range []struct {
		name  string
		model Model
		count int
		names []string
	}{
		{
			name:  "filtered credentials exclude revoked devices",
			model: Model{RemoteDeviceCount: 2, RemoteDeviceNames: []string{"active phone"}},
			count: 1,
			names: []string{"active phone"},
		},
		{
			name:  "all credentials revoked",
			model: Model{RemoteDeviceCount: 2, RemoteDeviceNames: []string{}},
		},
		{
			name:  "unavailable credential names retain count",
			model: Model{RemoteDeviceCount: 2},
			count: 2,
		},
		{
			name: "loaded runtime empty overrides credentials",
			model: Model{
				RemoteDeviceCount: 2,
				RemoteDeviceNames: []string{"stale phone"},
				PhoneRuntime:      phoneremote.RuntimeState{DevicesLoaded: true},
			},
		},
		{
			name: "loaded runtime devices override credentials",
			model: Model{
				RemoteDeviceCount: 2,
				RemoteDeviceNames: []string{},
				PhoneRuntime: phoneremote.RuntimeState{
					DevicesLoaded: true,
					Devices:       []phoneremote.Device{{Name: "live phone"}},
				},
			},
			count: 1,
			names: []string{"live phone"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := phoneSnapshot(tt.model)
			if snapshot.DeviceCount != tt.count || len(snapshot.DeviceNames) != len(tt.names) {
				t.Fatalf("device snapshot = %d %v, want %d %v", snapshot.DeviceCount, snapshot.DeviceNames, tt.count, tt.names)
			}
			for i, name := range tt.names {
				if snapshot.DeviceNames[i] != name {
					t.Fatalf("device names = %v, want %v", snapshot.DeviceNames, tt.names)
				}
			}
		})
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
