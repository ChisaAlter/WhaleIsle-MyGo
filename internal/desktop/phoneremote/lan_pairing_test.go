package phoneremote

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"testing"
	"time"
)

// syntheticOfferURL builds the pairing URL the native gateway emits: an
// https URL whose fragment carries a v2 offer with a future expiry.
func syntheticOfferURL(expires time.Time) string {
	offer := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(
		`{"v":2,"serverId":"lan-fixture","daemonPublicKeyB64":"fictional","authBootstrap":{"version":1,"pairingToken":"fictional-only","expiresAtMs":%d}}`,
		expires.UnixMilli())))
	return "https://192.168.1.10:3180/#offer=" + offer
}

// A listening LAN gateway has no relay connection, so its offer must still make
// the runtime ready (native remote.js:344-360 builds pairingUrl for both
// transports and only flags relayDown in relay mode). Relay mode stays strict.
func TestListeningLanDaemonPairsWithoutRelay(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mode       string
		wantReady  bool
		wantStatus string
	}{
		{name: "lan", mode: "lan", wantReady: true, wantStatus: "ready"},
		{name: "relay", mode: "relay", wantStatus: "relay-offline"},
		{name: "missing mode is relay", mode: "", wantStatus: "relay-offline"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, w := io.Pipe()
			child := &fixtureChild{output: r, writer: w, done: make(chan struct{})}
			m := NewRuntime()
			m.stopTimeout = time.Millisecond
			m.spawn = func(Resources) (Child, error) { return child, nil }
			m.resources = func(string, string) (Resources, error) { return Resources{}, nil }
			cfg := map[string]any{"remoteEnabled": true}
			if tc.mode != "" {
				cfg["remoteMode"] = tc.mode
			}
			m.Start("fixture", "fixture", cfg)
			defer m.Stop()
			url := syntheticOfferURL(time.Now().Add(time.Minute))
			go fmt.Fprintf(w, `{"kind":"snapshot","listening":true,"relayConnected":false,"url":%q}`+"\n", url)
			var s RuntimeState
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				s = m.Snapshot()
				if s.Status != "starting" && s.Status != "offline" {
					break
				}
				time.Sleep(time.Millisecond)
			}
			if s.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q (snapshot %+v)", s.Status, tc.wantStatus, s)
			}
			if !s.Listening || s.RelayConnected {
				t.Fatalf("snapshot flags = listening %v relay %v", s.Listening, s.RelayConnected)
			}
			if tc.wantReady {
				if s.PairingURL != url {
					t.Fatalf("PairingURL = %q, want the LAN offer", s.PairingURL)
				}
				if s.ExpiresAt.IsZero() {
					t.Fatal("LAN offer expiry was not recorded")
				}
			} else if s.PairingURL != "" {
				t.Fatalf("relay mode without relay exposed offer %q", s.PairingURL)
			}
		})
	}
}

// Refresh carries a transport change without restarting the daemon.
func TestRefreshRemembersLanMode(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()
	child := &fixtureChild{output: r, writer: w, done: make(chan struct{})}
	m := NewRuntime()
	m.stopTimeout = time.Millisecond
	m.spawn = func(Resources) (Child, error) { return child, nil }
	m.resources = func(string, string) (Resources, error) { return Resources{}, nil }
	m.Start("fixture", "fixture", map[string]any{"remoteEnabled": true, "remoteMode": "lan"})
	defer m.Stop()
	m.mu.Lock()
	started := m.lanMode
	m.mu.Unlock()
	if !started {
		t.Fatal("Start did not remember lan mode")
	}
	m.Refresh(map[string]any{"remoteEnabled": true, "remoteMode": "relay"})
	m.mu.Lock()
	refreshed := m.lanMode
	m.mu.Unlock()
	if refreshed {
		t.Fatal("Refresh kept lan mode after switching to relay")
	}
	if !bytes.Contains(child.input.Bytes(), []byte(`"action":"refresh"`)) {
		t.Fatal("refresh control not sent")
	}
}
