package phoneremote

import (
	"testing"

	"whaleisle/internal/config"
)

func TestAssessDisabledWhenPairingOff(t *testing.T) {
	s := Snapshot{
		RemoteEnabled:  false,
		RemoteMode:     "lan",
		RemotePort:     6767,
		RemoteBindAddress: "0.0.0.0",
		HasRemoteToken: true,
		MuxReady:       true,
	}
	r, missing := Assess(s)
	if r != ReadinessDisabled {
		t.Fatalf("Readiness = %q", r)
	}
	if len(missing) == 0 {
		t.Fatal("expected missing remoteEnabled")
	}
}

func TestAssessIncompleteWithoutToken(t *testing.T) {
	s := Snapshot{
		RemoteEnabled:     true,
		RemoteMode:        "lan",
		RemotePort:        6767,
		RemoteBindAddress: "192.168.1.10",
		HasRemoteToken:    false,
		MuxReady:          true,
	}
	r, missing := Assess(s)
	if r != ReadinessIncomplete {
		t.Fatalf("Readiness = %q", r)
	}
	found := false
	for _, m := range missing {
		if m == "remoteToken" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing=%v want remoteToken", missing)
	}
}

func TestAssessReadyLAN(t *testing.T) {
	s := Snapshot{
		RemoteEnabled:     true,
		RemoteMode:        "lan",
		RemotePort:        6767,
		RemoteBindAddress: "192.168.1.10",
		HasRemoteToken:    true,
		MuxReady:          true,
	}
	r, missing := Assess(s)
	if r != ReadinessReady {
		t.Fatalf("Readiness = %q missing=%v", r, missing)
	}
}

func TestAssessRelayNeedsRelayToken(t *testing.T) {
	s := Snapshot{
		RemoteEnabled:       true,
		RemoteMode:          "relay",
		HasRemoteToken:      true,
		HasRemoteRelayToken: false,
		MuxReady:            true,
	}
	r, missing := Assess(s)
	if r != ReadinessIncomplete {
		t.Fatalf("Readiness = %q", r)
	}
	found := false
	for _, m := range missing {
		if m == "remoteRelayToken" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing=%v want remoteRelayToken", missing)
	}
}

func TestConnectURLPrefersAppBase(t *testing.T) {
	s := Snapshot{
		RemoteMode:       "relay",
		RemoteAppBaseURL: "https://app.example.com/dshd/",
		RemotePort:       6767,
		RemoteBindAddress: "192.168.1.10",
	}
	got := ConnectURL(s)
	if got != "https://app.example.com/dshd" {
		t.Fatalf("ConnectURL = %q", got)
	}
}

func TestConnectURLLanNonLoopback(t *testing.T) {
	s := Snapshot{
		RemoteMode:        "lan",
		RemotePort:        6767,
		RemoteBindAddress: "192.168.1.10",
		RemoteLanTLS:      true,
	}
	got := ConnectURL(s)
	if got != "https://192.168.1.10:6767" {
		t.Fatalf("ConnectURL = %q", got)
	}
}

func TestConnectURLEmptyForLoopbackLAN(t *testing.T) {
	s := Snapshot{
		RemoteMode:        "lan",
		RemotePort:        6767,
		RemoteBindAddress: "127.0.0.1",
	}
	if got := ConnectURL(s); got != "" {
		t.Fatalf("ConnectURL = %q, want empty", got)
	}
}

func TestFromConfig(t *testing.T) {
	cfg := config.Defaults()
	cfg.RemoteEnabled = true
	cfg.RemoteMode = "lan"
	cfg.RemoteBindAddress = "192.168.1.20"
	cfg.RemotePort = 7000
	cfg.RemoteAppBaseUrl = ""
	s := FromConfig(cfg, true, false, true, 2)
	if s.Readiness != ReadinessReady {
		t.Fatalf("Readiness = %q missing=%v", s.Readiness, s.Missing)
	}
	if s.ConnectURL != "http://192.168.1.20:7000" {
		t.Fatalf("ConnectURL = %q", s.ConnectURL)
	}
	if s.DeviceCount != 2 {
		t.Fatalf("DeviceCount = %d", s.DeviceCount)
	}
}
