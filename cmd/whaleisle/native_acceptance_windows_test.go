//go:build windows && nativeacceptance

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/harness"
)

// This opt-in check runs the real desktop startup and real Host, with the
// production view, in a hidden window. Its caller must create a fresh isolated
// data directory and explicitly select a verified runtime. It never imports
// installed credentials or sends a prompt.
var nativeAcceptanceError error

func stopNativeAcceptance() {
	state.mu.Lock()
	state.quitting = true
	sup := state.sup
	state.mu.Unlock()
	if sup != nil {
		sup.Stop()
	}
	mygo.App.Quit()
}

// MyGo requires App.Run on the process main goroutine; testing.T runs its body
// on a worker. TestMain keeps the production entry point on that required thread.
func TestMain(m *testing.M) {
	if os.Getenv("WHALEISLE_ACCEPTANCE_OUTPUT") != "" {
		nativeAcceptanceError = runNativeCoreRecoveryDesktop()
	}
	os.Exit(m.Run())
}

func TestNativeCoreRecoveryDesktop(t *testing.T) {
	if os.Getenv("WHALEISLE_ACCEPTANCE_OUTPUT") == "" {
		t.Fatal("explicit acceptance invocation required")
	}
	if nativeAcceptanceError != nil {
		t.Fatal(nativeAcceptanceError)
	}
}

func runNativeCoreRecoveryDesktop() error {
	if !isolatedDesktopRun() || os.Getenv("WHALEISLE_USER_DATA") == "" || os.Getenv("WHALEISLE_VENDOR") == "" {
		return fmt.Errorf("explicit isolated data and verified vendor are required")
	}
	out := os.Getenv("WHALEISLE_ACCEPTANCE_OUTPUT")
	if !filepath.IsAbs(out) {
		return fmt.Errorf("absolute acceptance output required")
	}
	result := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(100 * time.Second)
		for time.Now().Before(deadline) {
			state.mu.Lock()
			win, status := state.win, state.bootStatus
			state.mu.Unlock()
			if status.State == harness.Failed {
				result <- fmt.Errorf("isolated desktop Host failed: %v", status.Err)
				stopNativeAcceptance()
				return
			}
			if win != nil && status.State == harness.Ready {
				// Same route change as closing the launcher after its Host is ready.
				state.revealMainConversation()
				time.Sleep(700 * time.Millisecond)
				png, err := win.CapturePage()
				if err == nil {
					err = os.WriteFile(out, png, 0o600)
				}
				if err == nil {
					meta, _ := json.MarshalIndent(map[string]any{"host": "ready", "windowHidden": true, "view": "production mainView", "data": "fresh isolated profile", "modelRequests": 0}, "", "  ")
					err = os.WriteFile(out+".json", meta, 0o600)
				}
				result <- err
				stopNativeAcceptance()
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		result <- fmt.Errorf("isolated desktop readiness timeout")
		stopNativeAcceptance()
	}()
	main()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("desktop exited before readiness")
	}
}
