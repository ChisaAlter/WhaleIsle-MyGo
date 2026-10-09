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

func TestResolvePrefersPathThenNpx(t *testing.T) {
	dir := t.TempDir()
	node := dir + "/node"
	npx := dir + "/npx"
	for _, f := range []string{node, npx} {
		if err := os.WriteFile(f, nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	defer func(orig func(string) (string, error)) { lookPath = orig }(lookPath)

	lookPath = func(name string) (string, error) {
		if name == "dsh" {
			return "/usr/bin/dsh", nil
		}
		return "", os.ErrNotExist
	}
	p, err := Resolve(LaunchOptions{Host: "127.0.0.1", NodeBin: node}, 3080)
	if err != nil || p.Command != "/usr/bin/dsh" || p.Args[0] != "web" {
		t.Fatalf("PATH plan %+v %v", p, err)
	}

	lookPath = func(string) (string, error) { return "", os.ErrNotExist }
	p, err = Resolve(LaunchOptions{Host: "127.0.0.1", NodeBin: node, Version: "0.2.0-rc.2", SkipUserPlugins: true}, 3081)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--yes", "@deepseek-ai/dsh@0.2.0-rc.2", "web", "--host", "127.0.0.1", "--port", "3081", "--no-open", "--skip-user-plugins"}
	if p.Command != npx || strings.Join(p.Args, " ") != strings.Join(want, " ") {
		t.Fatalf("npx plan %+v", p)
	}
}

func TestChildEnvSetsHome(t *testing.T) {
	t.Setenv("ELECTRON_RUN_AS_NODE", "1")
	t.Setenv("DSH_HOME", "/elsewhere")
	env := strings.Join(childEnv("/data/dsh-home"), "\n")
	if strings.Contains(env, "ELECTRON_RUN_AS_NODE") || strings.Contains(env, "/elsewhere") || !strings.Contains(env, "DSH_HOME=/data/dsh-home") {
		t.Fatalf("env:\n%s", env)
	}
}
