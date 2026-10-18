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
	"time"
)

func TestBridgeReadinessFixture(t *testing.T) {
	key := strings.Repeat("a", 32)
	apiPID := 123
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := url.Parse("http://" + r.Host)
		cookie, err := r.Cookie("magpie_web_" + u.Port())
		if err != nil || cookie.Value != key || r.URL.Path != "/api/whalebridge/status" || r.URL.RawQuery != "" {
			t.Error("invalid local status request")
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"pid": apiPID, "version": "1.2.3"})
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "state.json")
	st := bridgeState{ID: "whalebridge", PID: 123, Version: "1.2.3", Ready: true, URL: srv.URL + "/?k=" + key}
	write := func() {
		b, _ := json.Marshal(st)
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	exited := make(chan error, 1)
	if err := waitBridgeReady(context.Background(), path, "1.2.3", 123, exited, srv.Client()); err != nil {
		t.Fatal(err)
	}
	apiPID = 456
	if err := waitBridgeReady(context.Background(), path, "1.2.3", 123, exited, srv.Client()); err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("mismatch failure: %v", err)
	}
	apiPID = 123
	st.Version = "9.0.0"
	write()
	if err := waitBridgeReady(context.Background(), path, "1.2.3", 123, exited, srv.Client()); err == nil {
		t.Fatal("accepted wrong version")
	}
	st.Version = "1.2.3"
	st.PID = 999
	write()
	if err := waitBridgeReady(context.Background(), path, "1.2.3", 123, exited, srv.Client()); err == nil {
		t.Fatal("accepted another PID")
	}
	st.PID = 123
	st.URL = "https://example.com/?k=" + key
	write()
	if err := waitBridgeReady(context.Background(), path, "1.2.3", 123, exited, srv.Client()); err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("unsafe URL: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := waitBridgeReady(ctx, path, "1.2.3", 123, exited, srv.Client()); err == nil {
		t.Fatal("timeout reported success")
	}
	exited <- nil
	if err := waitBridgeReady(context.Background(), path, "1.2.3", 123, exited, srv.Client()); err == nil {
		t.Fatal("exit reported success")
	}
}
