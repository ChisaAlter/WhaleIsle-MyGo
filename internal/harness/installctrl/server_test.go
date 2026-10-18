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
