package marketctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
	marketstore "whaleisle/internal/store/market"
)

func TestViewSnapshotNonBlockingDuringBeginFinishAndInvalidate(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "personal.json")
	store := marketstore.NewWithPersonalPath(historyPath)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	store.Attach(client, nil)

	coord := NewCoordinator(nil)
	coord.SetStore(store)
	coord.SetClient(client)

	// Hammer ViewSnapshot concurrently while BeginAttempt, finishAttempt, and Invalidate
	// perform disk receipt I/O to ensure Coordinator.mu is never held across fsync.
	var stop atomic.Bool
	var reads atomic.Int64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop.Load() {
			_ = coord.ViewSnapshot()
			reads.Add(1)
		}
	}()

	for i := 0; i < 6; i++ {
		op, _ := coord.BeginAttempt("example-plugin", nil, true)
		if op == nil {
			t.Fatalf("BeginAttempt %d failed: %+v", i, coord.ViewSnapshot().Value)
		}
		if i%2 == 0 {
			coord.finishAttempt(op, marketResult{outcome: "applied", accepted: true, message: "ok"}, true)
			coord.releaseAttempt(op)
		} else {
			coord.Invalidate()
			coord.releaseAttempt(op)
		}
	}
	stop.Store(true)
	wg.Wait()

	if reads.Load() == 0 {
		t.Fatal("concurrent ViewSnapshot reader made zero progress")
	}
	snap := coord.ViewSnapshot().Value
	if snap.InstallBusy || snap.WorkerBusy {
		t.Fatalf("coordinator remained busy after attempts settled: %+v", snap)
	}
}

func TestCancelInstallStatuses(t *testing.T) {
	for _, status := range []string{"cancelled", "too-late", "not-running"} {
		t.Run(status, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var env map[string]any
				_ = json.NewDecoder(r.Body).Decode(&env)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type":   "server-response",
					"rpcId":  env["rpcId"],
					"result": map[string]any{"ok": true, "value": map[string]any{"status": status}},
				})
			}))
			defer srv.Close()

			historyPath := filepath.Join(t.TempDir(), "personal.json")
			store := marketstore.NewWithPersonalPath(historyPath)
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			store.Attach(client, nil)

			coord := NewCoordinator(nil)
			coord.SetStore(store)
			coord.SetClient(client)
			op, _ := coord.BeginAttempt("demo-pkg", nil, true)
			if op == nil {
				t.Fatal("BeginAttempt returned nil")
			}
			defer coord.releaseAttempt(op)

			coord.CancelInstall()
			waitMarketFixture(t, func() bool {
				v := coord.ViewSnapshot().Value
				if status == "cancelled" {
					return !v.InstallBusy && v.InstallMsg != ""
				}
				return v.InstallMsg != ""
			})
			if status == "cancelled" && op.Ctx.Err() != context.Canceled {
				t.Fatal("cancelled install did not cancel attempt context")
			}
			if status != "cancelled" {
				coord.Invalidate()
			}
		})
	}
}
