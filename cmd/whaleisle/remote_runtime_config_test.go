package main

import (
	"testing"

	"whaleisle/internal/config"
)

func TestBuildRemoteRuntimeConfigIncludesSavedTokensOnly(t *testing.T) {
	cfg := config.Config{
		RemoteEnabled:       true,
		RemoteMode:          "relay",
		RemotePort:          6767,
		RemoteBindAddress:   "127.0.0.1",
		RemoteLanTls:        false,
		RemoteAppBaseUrl:    "",
		RemoteRelayEndpoint: "ayase.cn:443",
		RemoteRelayUseTls:   true,
	}
	for _, tc := range []struct {
		name       string
		creds      remoteRuntimeCredentials
		wantLocal  string
		wantRelay  string
		absentKeys []string
	}{
		{name: "no tokens", creds: remoteRuntimeCredentials{}, absentKeys: []string{"remoteToken", "remoteRelayToken"}},
		{name: "blank tokens", creds: remoteRuntimeCredentials{Local: "  ", Relay: "\t"}, absentKeys: []string{"remoteToken", "remoteRelayToken"}},
		{name: "local only", creds: remoteRuntimeCredentials{Local: "local-token"}, wantLocal: "local-token", absentKeys: []string{"remoteRelayToken"}},
		{name: "relay only", creds: remoteRuntimeCredentials{Relay: "relay-token"}, wantRelay: "relay-token", absentKeys: []string{"remoteToken"}},
		{name: "both trimmed", creds: remoteRuntimeCredentials{Local: " local-token ", Relay: " relay-token\n"}, wantLocal: "local-token", wantRelay: "relay-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRemoteRuntimeConfig(cfg, tc.creds, []string{`C:\ws`}, "http://127.0.0.1:3080", "cookie-value")
			if local, _ := got["remoteToken"].(string); local != tc.wantLocal {
				t.Fatalf("remoteToken = %q, want %q", local, tc.wantLocal)
			}
			if relay, _ := got["remoteRelayToken"].(string); relay != tc.wantRelay {
				t.Fatalf("remoteRelayToken = %q, want %q", relay, tc.wantRelay)
			}
			for _, key := range tc.absentKeys {
				if _, ok := got[key]; ok {
					t.Fatalf("%s must be omitted when unset", key)
				}
			}
			// The rest of the daemon contract is unchanged.
			if got["remoteMode"] != "relay" || got["remotePort"] != 6767 || got["remoteBindAddress"] != "127.0.0.1" ||
				got["remoteRelayEndpoint"] != "ayase.cn:443" || got["remoteRelayUseTls"] != true ||
				got["harnessOrigin"] != "http://127.0.0.1:3080" || got["harnessCookie"] != "cookie-value" ||
				got["remoteGitEnabled"] != true {
				t.Fatalf("daemon config contract drifted: %#v", got)
			}
			if roots, ok := got["remoteGitRoots"].([]string); !ok || len(roots) != 1 || roots[0] != `C:\ws` {
				t.Fatalf("remoteGitRoots = %#v", got["remoteGitRoots"])
			}
		})
	}
}

func TestBuildRemoteRuntimeConfigNoGitRoots(t *testing.T) {
	got := buildRemoteRuntimeConfig(config.Config{RemoteEnabled: false, RemoteMode: "lan"}, remoteRuntimeCredentials{}, nil, "", "")
	if got["remoteGitEnabled"] != false || got["remoteMode"] != "lan" || got["remoteEnabled"] != false {
		t.Fatalf("config = %#v", got)
	}
}
