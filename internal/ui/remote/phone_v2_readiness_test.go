package remote

import (
	"testing"
	"whaleisle/internal/desktop/phoneremote"
)

func TestPhoneV2UsesObservedGatewayReadinessAndCurrentDeviceAuthority(t *testing.T) {
	base := Model{GatewayProtocol: "chisacode-v2", RemoteEnabled: true, RemoteMode: "relay", HarnessReady: false, MuxConnected: false, HasRemoteToken: false, HasRemoteRelayToken: false, RemoteDeviceCount: 4, RemoteDeviceNames: []string{"obsolete cache"}, PhoneRuntime: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: true, DevicesLoaded: true}}
	got := phoneSnapshot(base)
	if got.Readiness != phoneremote.ReadinessReady || len(got.Missing) != 0 || got.DeviceCount != 0 || len(got.DeviceNames) != 0 {
		t.Fatal("v2 depends on obsolete token/mux or stale device cache", got)
	}
	for _, state := range []struct {
		name             string
		status           string
		listening, relay bool
	}{
		{"starting", "starting", false, false}, {"offline-listener", "ready", false, true}, {"relay-down", "ready", true, false}, {"stopped", "stopped", false, false},
	} {
		t.Run(state.name, func(t *testing.T) {
			m := base
			m.PhoneRuntime.Status = state.status
			m.PhoneRuntime.Listening = state.listening
			m.PhoneRuntime.RelayConnected = state.relay
			snap := phoneSnapshot(m)
			if snap.Readiness == phoneremote.ReadinessReady || len(snap.Missing) != 0 {
				t.Fatal("configuration guessed a ready v2 daemon", snap)
			}
		})
	}
	base.RemoteEnabled = false
	if phoneSnapshot(base).Readiness != phoneremote.ReadinessDisabled {
		t.Fatal("disabled v2 gateway appears ready")
	}
}
