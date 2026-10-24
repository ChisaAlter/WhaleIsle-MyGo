package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/desktop/phoneremote"
)

func TestWritePhoneDiagRecordsOfferShapeWithoutToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WHALEISLE_USER_DATA", dir)
	t.Setenv(PhoneDiagEnv, "1")
	token := "fictional-pairing-token-do-not-persist"
	expires := time.Now().Add(5 * time.Minute)
	offer := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(
		`{"v":2,"serverId":"srv_fixture","daemonPublicKeyB64":"fictional","authBootstrap":{"version":1,"pairingToken":%q,"expiresAtMs":%d},"relay":{"endpoint":"ayase.cn:443","useTls":false}}`,
		token, expires.UnixMilli())))
	url := "http://127.0.0.1:3180/#offer=" + offer
	s := &mainState{}
	s.writePhoneDiag(phoneremote.RuntimeState{
		Status: "ready", Listening: true, RelayConnected: false, PairingURL: url,
		ExpiresAt: expires, PairingImage: nil,
	})
	path := filepath.Join(dir, "runtime", "remote-pairing.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("diag not written: %v", err)
	}
	if strings.Contains(string(raw), token) || strings.Contains(string(raw), offer) {
		t.Fatal("diag persisted the pairing token")
	}
	var d phoneDiag
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if d.Status != "ready" || !d.Listening || d.RelayConnected || !d.HasPairingURL {
		t.Fatalf("diag flags = %+v", d)
	}
	if len(d.PairingURLHash) != 64 || d.Scheme != "http" || d.Host != "127.0.0.1" || d.Port != "3180" {
		t.Fatalf("diag url shape = %+v", d)
	}
	if d.FragmentPrefix != "#offer=" || d.OfferVersion != 2 || d.OfferExpiresMs != expires.UnixMilli() {
		t.Fatalf("diag offer shape = %+v", d)
	}
	if d.RelayEndpoint != "ayase.cn:443" || d.RelayUseTLS {
		t.Fatalf("diag relay = %q tls=%v", d.RelayEndpoint, d.RelayUseTLS)
	}
	if d.ExpiresAt == "" {
		t.Fatal("diag missing expiry")
	}
}

func TestWritePhoneDiagDisabledWritesNothing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WHALEISLE_USER_DATA", dir)
	os.Unsetenv(PhoneDiagEnv)
	s := &mainState{}
	s.writePhoneDiag(phoneremote.RuntimeState{Status: "ready", Listening: true})
	if _, err := os.Stat(filepath.Join(dir, "runtime", "remote-pairing.json")); err == nil {
		t.Fatal("diag written without opt-in env")
	}
}
