package harness

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain lets the test binary stand in for dsh: with FAKE_DSH set it
// serves the launch-token handshake on the --port it is given.
func TestMain(m *testing.M) {
	switch os.Getenv("FAKE_DSH") {
	case "serve":
		fakeDsh()
		return
	case "crash":
		fmt.Println("boom")
		os.Exit(3)
	case "plugintree":
		for _, a := range os.Args {
			if a == "--skip-user-plugins" {
				fakeDsh()
				return
			}
		}
		fmt.Println("plugin tree failed to load")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

func fakeDsh() {
	var host, port string
	for i, a := range os.Args {
		switch a {
		case "--host":
			host = os.Args[i+1]
		case "--port":
			port = os.Args[i+1]
		}
	}
	l, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") == "secret" {
			http.SetCookie(w, &http.Cookie{Name: "dsh-auth-x", Value: "v1.ok", HttpOnly: true})
			w.Header().Set("Location", "./")
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	fmt.Printf("dsh web: http://%s/?token=secret\n", l.Addr())
	http.Serve(l, mux)
}

func supervisorFor(t *testing.T, mode string, opts Options) *Supervisor {
	t.Helper()
	t.Setenv("FAKE_DSH", mode)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	opts.Launch = LaunchOptions{DshBin: exe, Host: "127.0.0.1", Port: 47900, Home: t.TempDir()}
	return New(opts)
}

func waitFor(t *testing.T, s *Supervisor, want State) Status {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.Status(); st.State == want {
			return st
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("state %v, want %v; logs:\n%s", s.Status().State, want, strings.Join(s.Logs(), "\n"))
	return Status{}
}

func TestStartRedeemsCookieAndStops(t *testing.T) {
	s := supervisorFor(t, "serve", Options{})
	s.Start()
	st := waitFor(t, s, Ready)
	if st.Endpoint.Cookie != "dsh-auth-x=v1.ok" || !strings.HasPrefix(st.Endpoint.Origin, "http://127.0.0.1:") {
		t.Fatalf("endpoint %+v", st.Endpoint)
	}
	for _, line := range s.Logs() {
		if strings.Contains(line, "secret") {
			t.Fatalf("launch token leaked into the log: %q", line)
		}
	}
	s.Stop()
	if st := s.Status(); st.State != Stopped {
		t.Fatalf("after Stop: %v", st.State)
	}
	if _, err := http.Get(st.Endpoint.Origin); err == nil {
		t.Fatal("child still serving after Stop")
	}
}

func TestCrashRestartsThenFails(t *testing.T) {
	s := supervisorFor(t, "crash", Options{AutoRestart: true, MaxAttempts: 2, BaseDelay: 10 * time.Millisecond})
	s.Start()
	st := waitFor(t, s, Failed)
	if st.Attempt != 2 || st.Err == nil {
		t.Fatalf("status %+v", st)
	}
	restarts := 0
	for _, line := range s.Logs() {
		if strings.HasPrefix(line, "restart ") {
			restarts++
		}
	}
	if restarts != 2 {
		t.Fatalf("%d restarts; logs:\n%s", restarts, strings.Join(s.Logs(), "\n"))
	}
}

func TestPluginTreeRetrySkipsUserPlugins(t *testing.T) {
	s := supervisorFor(t, "plugintree", Options{})
	s.Start()
	st := waitFor(t, s, Ready)
	if st.Endpoint.Cookie != "dsh-auth-x=v1.ok" {
		t.Fatalf("endpoint %+v", st.Endpoint)
	}
	found := false
	for _, line := range s.Logs() {
		if strings.Contains(line, "--skip-user-plugins") || strings.Contains(line, "retrying with --skip-user-plugins") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected skip retry; logs:\n%s", strings.Join(s.Logs(), "\n"))
	}
	s.Stop()
}

func TestParseReady(t *testing.T) {
	url, ok := ParseReady("dsh web: http://127.0.0.1:3080/?token=abc")
	if !ok || url != "http://127.0.0.1:3080/?token=abc" {
		t.Fatalf("got %q %v", url, ok)
	}
	if _, ok := ParseReady("compiling profile"); ok {
		t.Fatal("matched a non-ready line")
	}
}

func TestRedeemWithoutTokenIsOpen(t *testing.T) {
	ep, err := Redeem(context.Background(), http.DefaultClient, "http://127.0.0.1:1/")
	if err != nil || ep.Cookie != "" || ep.Origin != "http://127.0.0.1:1" {
		t.Fatalf("got %+v %v", ep, err)
	}
}
