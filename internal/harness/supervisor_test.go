package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/paths"
)

// TestMain lets the test binary stand in for dsh: with FAKE_DSH set it
// serves the launch-token handshake on the --port it is given.
func TestMain(m *testing.M) {
	switch os.Getenv("FAKE_DSH") {
	case "hold":
		time.Sleep(time.Hour)
		return
	case "serve", "flap":
		fakeDsh()
		return
	case "leak":
		startStdoutLeaker()
		fakeDsh()
		return
	case "leaker":
		time.Sleep(time.Hour)
		return
	case "earlyparent":
		exe, err := os.Executable()
		if err != nil {
			os.Exit(4)
		}
		gc := exec.Command(exe)
		gc.Env = append(os.Environ(), "FAKE_DSH=leaker")
		if err := gc.Start(); err != nil {
			os.Exit(5)
		}
		if pidFile := os.Getenv("FAKE_DSH_GRANDCHILD_PID"); pidFile != "" {
			_ = os.WriteFile(pidFile, []byte(strconv.Itoa(gc.Process.Pid)), 0o600)
		}
		os.Exit(2)
	case "crash":
		fmt.Println("boom")
		os.Exit(3)
	case "trailinglogs":
		fmt.Println("trailing line one")
		fmt.Println("trailing line two")
		fmt.Print("plugin tree failed to load\n")
		os.Exit(2)
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
	release, err := isolateProfile()
	if err != nil {
		fmt.Fprintln(os.Stderr, "isolate test profile:", err)
		os.Exit(1)
	}
	code := m.Run()
	release()
	os.Exit(code)
}

// isolateProfile points the tests at a throwaway profile before any of them
// run. Every capability check here hashes artifacts through
// paths.UserDataDir(), which falls back to the user's real data directory, so a
// test that does not isolate the profile would write the machine it runs on.
func isolateProfile() (release func(), err error) {
	if strings.TrimSpace(os.Getenv(paths.EnvUserData)) != "" {
		return func() {}, nil
	}
	dir, err := os.MkdirTemp("", "whaleisle-harness-profile-")
	if err != nil {
		return nil, err
	}
	if err = os.Setenv(paths.EnvUserData, dir); err != nil {
		return nil, err
	}
	return func() { _ = os.RemoveAll(dir) }, nil
}

func fakeDsh() {
	if marker := os.Getenv("FAKE_DSH_WORKSPACE_MARKER"); marker != "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("cwd:", err)
			os.Exit(1)
		}
		raw, _ := json.Marshal(map[string]string{"cwd": cwd})
		if err := os.WriteFile(marker, raw, 0600); err != nil {
			fmt.Println("cwd marker:", err)
			os.Exit(1)
		}
	}
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
			if os.Getenv("FAKE_DSH") == "flap" {
				// Allow the successful handshake to complete before dying online.
				time.AfterFunc(200*time.Millisecond, func() { os.Exit(3) })
			}
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
	stopStart := time.Now()
	s.Stop()
	// Reaching stopGrace means the first terminate left the child running.
	if took := time.Since(stopStart); took >= stopGrace {
		t.Fatalf("Stop took %s; the first terminate did not end the child", took)
	}
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

func assertInstallControlActive(t *testing.T, srv *installctrl.Server) {
	t.Helper()
	var baseURL, token string
	for _, kv := range srv.Env() {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if k == installctrl.EnvURL {
			baseURL = v
		}
		if k == installctrl.EnvToken {
			token = v
		}
	}
	if baseURL == "" || token == "" {
		t.Fatalf("installctrl env missing credentials: %v", srv.Env())
	}
	for _, tc := range []struct {
		path string
		body string
	}{
		{path: "/install", body: `{"spec":"github:acme/widget"}`},
		{path: "/desktop/project", body: `{"action":"ping"}`},
	} {
		req, err := http.NewRequest(http.MethodPost, baseURL+tc.path, bytes.NewBufferString(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s request failed: %v", tc.path, err)
		}
		var decoded map[string]any
		decodeErr := json.NewDecoder(resp.Body).Decode(&decoded)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || decodeErr != nil {
			t.Fatalf("%s status=%d decodeErr=%v body=%v", tc.path, resp.StatusCode, decodeErr, decoded)
		}
		if decoded["ok"] != true {
			t.Fatalf("%s expected ok=true, got %v", tc.path, decoded)
		}
	}
}

func TestInstallControlSurvivesAutoRestartAndManualRetry(t *testing.T) {
	makeInstall := func() *installctrl.Server {
		return installctrl.New(installctrl.Options{
			InstallContext: func(ctx context.Context, spec string, _ installctrl.InstallOpts) installctrl.Result {
				if err := ctx.Err(); err != nil {
					return installctrl.Result{OK: false, Spec: spec, Error: err.Error()}
				}
				return installctrl.Result{OK: true, Spec: spec}
			},
			Project: func(ctx context.Context, _ map[string]any) map[string]any {
				if err := ctx.Err(); err != nil {
					return map[string]any{"ok": false, "error": err.Error()}
				}
				return map[string]any{"ok": true}
			},
		})
	}

	t.Run("PluginTreeAutoRestart", func(t *testing.T) {
		srv := makeInstall()
		s := supervisorFor(t, "plugintree", Options{Install: srv})
		defer s.Stop()
		s.Start()
		waitFor(t, s, Ready)
		assertInstallControlActive(t, srv)
	})

	t.Run("CrashFailureThenManualStart", func(t *testing.T) {
		srv := makeInstall()
		s := supervisorFor(t, "crash", Options{
			AutoRestart: true,
			MaxAttempts: 1,
			BaseDelay:   10 * time.Millisecond,
			Install:     srv,
		})
		defer s.Stop()
		s.Start()
		waitFor(t, s, Failed)

		t.Setenv("FAKE_DSH", "serve")
		s.Start()
		waitFor(t, s, Ready)
		assertInstallControlActive(t, srv)
	})
}
