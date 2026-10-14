package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func waitStore(t *testing.T, s *Store) Snapshot {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot()
		if !v.Busy {
			return v
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("store did not finish")
	return Snapshot{}
}
func TestMachineLifecycleAndFailureFeedback(t *testing.T) {
	machines := []map[string]any{}
	current := ""
	connected := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "session=fixture" {
			t.Error("missing session cookie")
		}
		w.Header().Set("Content-Type", "application/json")
		var b map[string]any
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&b)
		}
		switch r.URL.Path {
		case "/api/dsh-remote/machines":
			if r.Method == "POST" {
				switch b["action"] {
				case "add":
					b["id"] = "one"
					machines = append(machines, b)
				case "update":
					machines[0] = b
				case "delete":
					machines = nil
					current = ""
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"machines": machines, "currentId": current})
		case "/api/dsh-remote/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"connected": connected})
		case "/api/dsh-remote/current":
			current, _ = b["id"].(string)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/dsh-remote/connect":
			_, _ = w.Write([]byte(`{"ok":false,"error":"fixture connection refused"}`))
		default:
			t.Error("unexpected route", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL, Cookie: "session=fixture"}}, nil)
	s.Refresh()
	if v := waitStore(t, s); v.Error != "" || !v.Loaded {
		t.Fatal(v)
	}
	d := Draft{Name: "Dev", Host: "fixture.invalid", Port: "22", Username: "developer", Workspace: "/work"}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	v := waitStore(t, s)
	if len(v.Machines) != 1 || v.CurrentID != "" {
		t.Fatal("save must not activate", v)
	}
	s.Select("one")
	if v = waitStore(t, s); v.CurrentID != "one" {
		t.Fatal(v)
	}
	d.ID = "one"
	d.Name = "Renamed"
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	if v = waitStore(t, s); v.Machines[0].Name != "Renamed" {
		t.Fatal(v)
	}
	s.Connect()
	if v = waitStore(t, s); !strings.Contains(v.Error, "fixture connection refused") || v.Connected || v.Busy {
		t.Fatal(v)
	}
	s.Select("")
	if v = waitStore(t, s); v.CurrentID != "" || len(v.Machines) != 1 {
		t.Fatal(v)
	}
	s.Remove("one")
	if v = waitStore(t, s); len(v.Machines) != 0 || v.CurrentID != "" {
		t.Fatal(v)
	}
}
func TestEditPreservesAuthenticationAndRejectsInvalidInput(t *testing.T) {
	var m Machine
	if err := json.Unmarshal([]byte(`{"id":"one","host":"example.invalid","port":22,"username":"dev","hostKeyMode":"verify","useAgent":true,"keyboardInteractive":true,"passphrase":"fixture-only","credentialBackend":"dpapi","proxy":{"host":"proxy.invalid","username":"jump","passwordSet":true}}`), &m); err != nil {
		t.Fatal(err)
	}
	s := New()
	s.view.Machines = []Machine{m}
	d := Draft{ID: "one", Host: "example.invalid", Port: "22", Username: "dev"}
	b, err := s.payload(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"hostKeyMode", "useAgent", "keyboardInteractive", "passphrase", "credentialBackend", "proxy"} {
		if string(b[k].(json.RawMessage)) != string(m.raw[k]) {
			t.Fatalf("changed authentication field %s", k)
		}
	}
	if _, ok := b["password"]; ok {
		t.Fatal("must not introduce a password")
	}
	for _, port := range []string{"22abc", "0", "65536", "-1", "x"} {
		d.Port = port
		if _, err := s.payload(d); err == nil {
			t.Fatal("accepted invalid port", port)
		}
	}
	d.Port = "22"
	d.ID = "missing"
	if _, err := s.payload(d); err == nil {
		t.Fatal("accepted stale editor")
	}
}
func TestOldEndpointCannotOverwriteNewEndpoint(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "machines") {
			close(entered)
			<-release
			_, _ = w.Write([]byte(`{"machines":[],"currentId":"old"}`))
		} else {
			_, _ = w.Write([]byte(`{"connected":true}`))
		}
	}))
	defer server.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	s.Refresh()
	<-entered
	s.Attach(nil, nil)
	close(release)
	// A synchronous request verifies transport errors do not dereference a nil client.
	if err := request(context.Background(), nil, "GET", "status", nil, nil); err == nil {
		t.Fatal("expected not ready")
	}
	time.Sleep(20 * time.Millisecond)
	if v := s.Snapshot(); v.CurrentID != "" || v.Connected || v.Loaded {
		t.Fatal("stale response applied", v)
	}
}
