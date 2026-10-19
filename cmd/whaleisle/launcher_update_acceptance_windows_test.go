//go:build windows && sessionrecoveryacceptance

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	desktopupdate "whaleisle/internal/desktop/update"
)

// probeLauncherUpdateResponsiveness runs only in the caller's isolated native
// process. It proves main-queue progress during real loopback HTTP, not OS clicks.
func probeLauncherUpdateResponsiveness(win *mygo.Window) (map[string]any, error) {
	proof := map[string]any{"scope": "native main-thread queue and real loopback HTTP; no OS clicks"}
	if win == nil {
		return proof, fmt.Errorf("owned native window required")
	}
	state.mu.Lock()
	busy, quitting := state.launcherUpdateCheckBusy, state.quitting
	state.mu.Unlock()
	if busy || quitting {
		return proof, fmt.Errorf("owned native state is already checking updates or quitting")
	}
	requested, release := make(chan struct{}, 2), make(chan struct{})
	var requests atomic.Int32
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		requested <- struct{}{}
		<-release
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(desktopupdate.Feed{Version: buildVersion, URL: "https://example.invalid/desktop.zip"})
	}))
	defer server.Close()
	defer unblock()
	old, hadOld := os.LookupEnv(desktopupdate.EnvFeedURL)
	if err := os.Setenv(desktopupdate.EnvFeedURL, server.URL); err != nil {
		return proof, err
	}
	defer func() {
		if hadOld {
			_ = os.Setenv(desktopupdate.EnvFeedURL, old)
		} else {
			_ = os.Unsetenv(desktopupdate.EnvFeedURL)
		}
	}()
	callbackReturned, mainQueuePing := make(chan struct{}), make(chan struct{})
	started := time.Now()
	win.Update(func() { state.checkLauncherUpdate(win); close(callbackReturned) })
	win.Update(func() { close(mainQueuePing) })
	wait := func(ch <-chan struct{}, name string) error {
		select {
		case <-ch:
			proof[name] = true
			return nil
		case <-time.After(3 * time.Second):
			return fmt.Errorf("%s did not complete while HTTP response was gated", name)
		}
	}
	if err := wait(requested, "loopbackRequestStarted"); err != nil {
		return proof, err
	}
	if err := wait(callbackReturned, "callbackReturnedBeforeResponse"); err != nil {
		return proof, err
	}
	if err := wait(mainQueuePing, "mainQueuePingBeforeResponse"); err != nil {
		return proof, err
	}
	proof["queueProgressMs"] = time.Since(started).Milliseconds()
	state.mu.Lock()
	busy = state.launcherUpdateCheckBusy
	state.mu.Unlock()
	if !busy {
		return proof, fmt.Errorf("busy cleared before gated HTTP response")
	}
	proof["busyDuringGatedResponse"] = true
	unblock()
	deadline := time.Now().Add(5 * time.Second)
	for {
		state.mu.Lock()
		busy, message, available := state.launcherUpdateCheckBusy, state.launcherUpdate, state.launcherUpdateAvailable
		state.mu.Unlock()
		if !busy {
			proof["busyClearedAfterResponse"], proof["result"], proof["requestCount"] = true, message, requests.Load()
			if !strings.Contains(message, string(desktopupdate.StatusUpToDate)) || available {
				return proof, fmt.Errorf("unexpected native update result: %q, available=%v", message, available)
			}
			if requests.Load() != 1 {
				return proof, fmt.Errorf("unexpected request count: %d", requests.Load())
			}
			return proof, nil
		}
		if time.Now().After(deadline) {
			return proof, fmt.Errorf("native queued result did not clear busy")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
