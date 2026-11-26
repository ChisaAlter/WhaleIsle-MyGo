package assistantctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/store/session"
)

func TestContactRoomRuntimesClearsBusyOnTransientNotCurrent(t *testing.T) {
	entered := make(chan struct{}, 4)
	release := make(chan struct{})
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		n := calls.Add(1)
		if n == 1 {
			entered <- struct{}{}
			<-release
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": req.RPCID,
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"room": map[string]any{"id": "room-1", "sessionId": "sess-room-1"},
				},
			},
		})
	}))
	defer srv.Close()

	lctx, lcancel := context.WithCancel(context.Background())
	defer lcancel()
	ep := harness.Endpoint{Origin: srv.URL}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	sessions := session.New()
	sessions.Attach(client, nil)
	sessions.Select("sess-room-1")

	asst := assiststore.New()
	asst.Attach(nil)

	svc := NewService(HostDeps{
		LifecycleContext: func() context.Context { return lctx },
		CurrentClient:    func() *dsh.Client { return client },
		BootEndpoint:     func() (harness.Endpoint, harness.State, int) { return ep, harness.Ready, 1 },
		Sessions:         func() *session.Store { return sessions },
		Window:           func() any { return "win" },
		WindowUpdate:     func(any, func()) {},
	})
	t.Cleanup(svc.SuspendAll)
	svc.AttachStores(asst, nil, nil)

	catalog := pluginrpc.DshbotContactCatalog{
		DshbotBotCatalog: pluginrpc.DshbotBotCatalog{
			Items: []pluginrpc.DshbotBotItem{{Kind: "room", ID: "room-1", SessionID: "sess-room-1"}},
		},
	}

	// First call launches in-flight poll with busy=true.
	_ = svc.contactRoomRuntimes(ep, catalog)
	<-entered

	// Transiently change selected session so current() returns false when RPC finishes,
	// without tearing down the contacts scope (simulating a momentary ownership flicker).
	sessions.Select("other-session")
	close(release)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		svc.assistantGroupLive.contacts.mu.Lock()
		busy := svc.assistantGroupLive.contacts.busy
		cancelFn := svc.assistantGroupLive.contacts.cancel
		svc.assistantGroupLive.contacts.mu.Unlock()
		if !busy && cancelFn == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	svc.assistantGroupLive.contacts.mu.Lock()
	stillBusy := svc.assistantGroupLive.contacts.busy
	svc.assistantGroupLive.contacts.mu.Unlock()
	if stillBusy {
		t.Fatal("contactRoomRuntimes remained permanently wedged with busy=true after !current()")
	}

	// Switching back to sess-room-1 must be able to start a fresh poll.
	sessions.Select("sess-room-1")
	_ = svc.contactRoomRuntimes(ep, catalog)
	for time.Now().Before(deadline) {
		if calls.Load() >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected contactRoomRuntimes to restart poll after recovery, calls=%d", calls.Load())
	}
}

func TestWhaleTaskPollContextCancelRaceFree(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": req.RPCID,
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"tasks": []map[string]any{{
						"id":    "t1",
						"state": "running",
						"effects": []map[string]any{{
							"state": "pending",
						}},
					}},
				},
			},
		})
	}))
	defer srv.Close()

	lctx, lcancel := context.WithCancel(context.Background())
	ep := harness.Endpoint{Origin: srv.URL}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	svc := NewService(HostDeps{
		LifecycleContext: func() context.Context { return lctx },
		CurrentClient:    func() *dsh.Client { return client },
		BootEndpoint:     func() (harness.Endpoint, harness.State, int) { return ep, harness.Ready, 1 },
		Window:           func() any { return "win" },
		WindowUpdate:     func(any, func()) {},
	})
	t.Cleanup(svc.SuspendAll)

	_ = svc.WhaleTasksModel(ep)
	time.Sleep(30 * time.Millisecond)
	lcancel()
	time.Sleep(20 * time.Millisecond)
}
