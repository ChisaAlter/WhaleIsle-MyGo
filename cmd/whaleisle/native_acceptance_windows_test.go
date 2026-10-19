//go:build windows && nativeacceptance

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
)

// This opt-in check runs the real desktop startup and real Host, with the
// production view, in a hidden window. Its caller must create a fresh isolated
// data directory and explicitly select a verified runtime. It never imports
// installed credentials or sends a prompt.
var nativeAcceptanceError error

// These reads exercise served namespaces used by the full desktop. Each result
// stays separate from window acceptance and never triggers install/run actions.
func nativeAcceptanceReadSmoke(client *dsh.Client) map[string]any {
	out := map[string]any{}
	for _, method := range []string{"settings/describe", "pluginManager/listBundles", "schedule/catalog"} {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		var value json.RawMessage
		err := client.Call(ctx, method, nil, &value)
		cancel()
		item := map[string]any{"pass": err == nil, "responseBytes": len(value)}
		if err != nil {
			item["error"] = err.Error()
		}
		out[method] = item
	}
	return out
}

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
		revealed := false
		requestedSelection := false
		selection := os.Getenv("WHALEISLE_ACCEPTANCE_SESSION")
		for time.Now().Before(deadline) {
			state.mu.Lock()
			win, status, sessions, muxReady := state.win, state.bootStatus, state.sessions, state.dshMuxReady
			state.mu.Unlock()
			if status.State == harness.Failed {
				result <- fmt.Errorf("isolated desktop Host failed: %v", status.Err)
				stopNativeAcceptance()
				return
			}
			if win != nil && status.State == harness.Ready {
				// Same route change as closing the launcher after its Host is ready.
				if !revealed {
					state.revealMainConversation()
					revealed = true
				}
				if sessions == nil {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				snap := sessions.Snapshot()
				if !muxReady || !snap.ClientReady || snap.ListLoading || !snap.WorkspaceReady {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				if snap.ListErr != "" {
					result <- fmt.Errorf("isolated session list failed: %s", snap.ListErr)
					stopNativeAcceptance()
					return
				}
				if selection != "" {
					if !requestedSelection {
						sessions.Select(selection)
						requestedSelection = true
						continue
					}
					if snap.HistoryError != "" || snap.GoalReadError != "" {
						result <- fmt.Errorf("isolated selected history or goal read failed")
						stopNativeAcceptance()
						return
					}
					if snap.SelectedID != selection || !snap.HistoryReady || snap.HistoryLoading {
						time.Sleep(100 * time.Millisecond)
						continue
					}
					if len(snap.Messages) == 0 || snap.PromptBusy {
						result <- fmt.Errorf("isolated persisted selection has no messages or is unexpectedly running")
						stopNativeAcceptance()
						return
					}
				}
				time.Sleep(700 * time.Millisecond)
				if strings.HasPrefix(sessions.Snapshot().LastAction, "workspace follow:") {
					result <- fmt.Errorf("workspace follow error remains in the ready desktop")
					stopNativeAcceptance()
					return
				}
				state.dshMu.Lock()
				client := state.dshClient
				state.dshMu.Unlock()
				if client == nil {
					result <- fmt.Errorf("ready desktop has no client")
					stopNativeAcceptance()
					return
				}
				reads := nativeAcceptanceReadSmoke(client)
				png, err := win.CapturePage()
				if err == nil {
					err = os.WriteFile(out, png, 0o600)
				}
				if err == nil {
					meta, _ := json.MarshalIndent(map[string]any{"host": "ready", "mux": "ready", "sessionList": "ready", "workspace": "ready", "historyReady": snap.HistoryReady, "messageCount": len(snap.Messages), "readSmoke": reads, "windowHidden": true, "view": "production mainView", "data": "isolated profile", "modelRequests": 0}, "", "  ")
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
