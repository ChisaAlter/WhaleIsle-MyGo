package components

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/desktop/bridgealerts"
)

func TestManagedBridgeAlertContractFixture(t *testing.T) {
	root := t.TempDir()
	key := strings.Repeat("a", 32)
	pid := 123
	claimCalls, opens := 0, 0
	statusPID := 123
	badClaim := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := url.Parse("http://" + r.Host)
		cookie, e := r.Cookie("magpie_web_" + u.Port())
		if e != nil || cookie.Value != key || r.URL.RawQuery != "" {
			t.Error("credential in URL or missing scoped cookie")
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/api/whalebridge/status":
			json.NewEncoder(w).Encode(map[string]any{"pid": statusPID, "version": "1.2.3"})
		case "/api/subscriptions/alerts/claim":
			claimCalls++
			if r.Method != "POST" || r.ContentLength != 0 {
				t.Error("invented claim method/body")
			}
			if badClaim {
				w.Write([]byte(`{"alerts":`))
				return
			}
			w.Write([]byte(`{"sequence":7,"alerts":[{"sequence":7,"alert":{"Name":"Fake","Kind":"expires","Credits":3,"ResetsAt":"2030-01-01T00:00:00Z"}}]}`))
		default:
			t.Error("invented endpoint")
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	os.MkdirAll(filepath.Join(root, "data"), 0700)
	os.WriteFile(filepath.Join(root, "installed.json"), []byte(`{"version":"1.2.3"}`), 0600)
	st := bridgeState{ID: "whalebridge", Version: "1.2.3", PID: pid, Ready: true, URL: srv.URL + "/?k=" + key}
	write := func() { b, _ := json.Marshal(st); os.WriteFile(filepath.Join(root, "data", "state.json"), b, 0600) }
	write()
	source := &bridgeAlertSource{root: root, client: srv.Client(), processIdentity: func(int, string) (string, error) { return "creation-one", nil }, open: func(raw string) error {
		if raw != st.URL {
			t.Error("management retarget")
		}
		opens++
		return nil
	}}
	m := bridgealerts.New(source)
	m.Poll()
	snap := m.Snapshot()
	if claimCalls != 1 || len(snap.Alerts) != 1 || !strings.Contains(snap.Alerts[0].Body, "3 次") {
		t.Fatalf("contract not consumed: %#v claims=%d", snap, claimCalls)
	}
	if e := m.Open(snap.Alerts[0].Ticket); e != nil || opens != 1 {
		t.Fatalf("open: %v", e)
	}
	st.PID = 456
	write()
	if e := m.Open(snap.Alerts[0].Ticket); e == nil || opens != 1 {
		t.Fatal("old PID opened new instance")
	}
	m.Poll()
	if claimCalls != 1 {
		t.Fatal("unverified status claimed")
	}
	statusPID = 456
	badClaim = true
	m.Poll()
	m.Poll()
	if claimCalls != 2 || !m.Snapshot().Paused {
		t.Fatal("uncertain response retried")
	}
	st.URL = "https://example.com/?k=" + key
	write()
	if _, e := source.Current(context.Background()); e == nil || strings.Contains(e.Error(), key) {
		t.Fatal("external/secret management address accepted")
	}
}

func TestBridgeAlertReceiptEnvelopeFixture(t *testing.T) {
	for _, body := range []string{`{}`, `{"sequence":1}`, `{"alerts":[]}`, `{"sequence":1,"alerts":null}`} {
		if _, err := decodeBridgeAlertReceipt([]byte(body)); err == nil {
			t.Fatalf("accepted unconfirmed receipt %s", body)
		}
	}
	got, err := decodeBridgeAlertReceipt([]byte(`{"sequence":0,"alerts":[]}`))
	if err != nil || got.Sequence != 0 || got.Alerts == nil {
		t.Fatalf("valid empty queue: %#v %v", got, err)
	}
}
