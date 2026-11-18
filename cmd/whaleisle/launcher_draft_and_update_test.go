package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

func TestLauncherInstallUpdateNonBlockingAndReentrancy(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	t.Setenv("DSH_HOME", t.TempDir())
	var requests atomic.Int32
	requested := make(chan struct{}, 4)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		requested <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(desktopupdate.Feed{Version: "9.9.9", URL: "https://example.invalid/desktop.zip"})
	}))
	defer server.Close()
	defer unblock()
	t.Setenv(desktopupdate.EnvFeedURL, server.URL)

	s := &mainState{}
	returned := make(chan struct{})
	go func() {
		s.launcherController().RunUpdateProbe(nil, "0.0.1", false)
		s.launcherController().RunUpdateProbe(nil, "0.0.1", false)
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(1 * time.Second):
		t.Fatal("RunUpdateProbe blocked caller while HTTP response was gated")
	}
	select {
	case <-requested:
	case <-time.After(2 * time.Second):
		t.Fatal("expected HTTP update request to start asynchronously")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("expected reentrancy guard to allow 1 in-flight request, got %d", got)
	}

	unblock()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		probe := s.launcherController().UpdateProbeState()
		if !probe.InstallBusy && probe.Available {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for async launcher install update probe to settle")
}
