package installctrl

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestInstallSuccess(t *testing.T) {
	var called atomic.Bool
	srv := New(Options{
		Install: func(spec string, opts InstallOpts) Result {
			called.Store(true)
			if spec != "github:acme/widget" {
				t.Fatalf("spec = %q", spec)
			}
			if len(opts.AllowBuilds) != 1 || opts.AllowBuilds[0] != "esbuild" {
				t.Fatalf("allowBuilds = %v", opts.AllowBuilds)
			}
			return Result{OK: true, Spec: spec}
		},
	})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	body, _ := json.Marshal(map[string]any{
		"spec":        "github:acme/widget",
		"allowBuilds": []string{"esbuild"},
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/install", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+srv.Token())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Spec != "github:acme/widget" {
		t.Fatalf("result = %+v", result)
	}
	if !called.Load() {
		t.Fatal("InstallFunc not called")
	}
}

func TestInstallUnauthorized(t *testing.T) {
	srv := New(Options{Install: StubOKInstall()})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/install", bytes.NewReader([]byte(`{"spec":"github:a/b"}`)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestInstallBadSpec(t *testing.T) {
	srv := New(Options{Install: StubOKInstall()})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	body := []byte(`{"spec":"npm:foo"}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/install", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestInstallSchedulesRestart(t *testing.T) {
	ch := make(chan struct{}, 1)
	srv := New(Options{
		Install:      StubOKInstall(),
		OnRestart:    func() { ch <- struct{}{} },
		RestartDelay: 20 * time.Millisecond,
	})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	body := []byte(`{"spec":"github:acme/widget"}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/install", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("OnRestart not called")
	}
}

func TestEnvVars(t *testing.T) {
	srv := New(Options{Install: StubOKInstall()})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()
	env := srv.Env()
	if len(env) != 2 {
		t.Fatalf("env = %v", env)
	}
}

func TestStopCancelsInflightInstallAndBlocksRestart(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	var restarts atomic.Int32
	srv := New(Options{
		InstallContext: func(ctx context.Context, spec string, _ InstallOpts) Result {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return Result{OK: true, Spec: spec}
		},
		OnRestart:    func() { restarts.Add(1) },
		RestartDelay: 15 * time.Millisecond,
	})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/install", bytes.NewReader([]byte(`{"spec":"github:acme/slow"}`)))
		req.Header.Set("Authorization", "Bearer "+srv.Token())
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("install handler did not start")
	}

	srv.Stop()

	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not cancel in-flight install context")
	}
	select {
	case <-reqDone:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP request goroutine did not exit")
	}

	time.Sleep(60 * time.Millisecond)
	if restarts.Load() != 0 {
		t.Fatalf("OnRestart fired %d times after Stop", restarts.Load())
	}
}

func TestRollbackFieldsMappedOnInstallAndDesktopPlugin(t *testing.T) {
	srv := New(Options{
		InstallContext: func(_ context.Context, spec string, _ InstallOpts) Result {
			return Result{
				OK:            false,
				Spec:          spec,
				Error:         "install failed; rollback failed",
				RolledBack:    true,
				RollbackError: "rollback failure detail",
			}
		},
	})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	for _, tc := range []struct {
		path string
		body string
	}{
		{path: "/install", body: `{"spec":"github:acme/widget"}`},
		{path: "/desktop/plugin", body: `{"action":"install","spec":"github:acme/widget"}`},
	} {
		req, _ := http.NewRequest(http.MethodPost, srv.URL()+tc.path, bytes.NewReader([]byte(tc.body)))
		req.Header.Set("Authorization", "Bearer "+srv.Token())
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		var got map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			resp.Body.Close()
			t.Fatalf("%s decode: %v", tc.path, err)
		}
		resp.Body.Close()
		if got["rolledBack"] != true {
			t.Fatalf("%s rolledBack = %#v, want true (body=%v)", tc.path, got["rolledBack"], got)
		}
		if got["rollbackError"] != "rollback failure detail" {
			t.Fatalf("%s rollbackError = %#v", tc.path, got["rollbackError"])
		}
	}
}
