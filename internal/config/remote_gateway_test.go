package config

import "testing"

func TestRemoteGatewayPreferencesValidationAndPreservation(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	old := s.Get()
	p := RemoteGatewayFrom(old)
	p.Mode = "lan"
	p.Port = "7788"
	p.BindAddress = "0.0.0.0"
	p.RelayEndpoint = "relay.example.com:443"
	if err := s.SaveRemoteGateway(RemoteGatewayFrom(old), p); err != nil {
		t.Fatal(err)
	}
	got := s.Get()
	if got.RemotePort != 7788 || got.RemoteMode != "lan" || got.RemoteBindAddress != "0.0.0.0" || got.RemoteRelayEndpoint != "relay.example.com:443" {
		t.Fatalf("bad gateway: %+v", got)
	}
	if got.RemoteEnabled != old.RemoteEnabled || got.Workspace != old.Workspace {
		t.Fatal("unrelated configuration changed")
	}
	if err := s.SaveRemoteGateway(RemoteGatewayFrom(old), p); err == nil {
		t.Fatal("stale editor accepted")
	}
	for _, bad := range []RemoteGatewayPreferences{{Mode: "wrong", Port: "7788", BindAddress: "127.0.0.1"}, {Mode: "ssh", Port: "7788", BindAddress: "127.0.0.1"}, {Mode: "lan", Port: "80", BindAddress: "127.0.0.1"}, {Mode: "lan", Port: "7788", BindAddress: "host.invalid"}, {Mode: "lan", Port: "7788", BindAddress: "127.0.0.1", RelayEndpoint: "https://user:secret@host/path"}} {
		if err := s.SaveRemoteGateway(RemoteGatewayFrom(got), bad); err == nil {
			t.Fatalf("invalid accepted: %+v", bad)
		}
	}
	if s.Get().RemotePort != 7788 {
		t.Fatal("invalid update mutated state")
	}
}
