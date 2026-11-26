package projectflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/dsh"
)

func TestIsTransientFollowErrorCoversRetryableConditions(t *testing.T) {
	for _, err := range []error{
		dsh.ErrCarrierLost,
		dsh.ErrStreamEnd,
		dsh.ErrStreamOverflow,
		dsh.ErrEndpointNotReady,
		errors.New("dsh mux: closed"),
		&dsh.TransportError{Op: "dial", Err: errors.New("reset")},
	} {
		if !isTransientFollowError(err) {
			t.Fatalf("expected %v to be classified transient", err)
		}
	}
	if isTransientFollowError(errors.New("permission denied")) {
		t.Fatal("non-transient error was classified transient")
	}
}

func TestFollowProcessRetriesTransientDialOpenAfterInitialSnapshot(t *testing.T) {
	var connNum atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := connNum.Add(1)
		if n == 2 {
			// Simulate transient dial/upgrade failure while Host mux is reconnecting.
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		var open dsh.ClientFrame
		if err := wsjson.Read(r.Context(), c, &open); err != nil {
			return
		}
		snapRaw, _ := json.Marshal(map[string]any{
			"type": "snapshot",
			"records": []map[string]any{
				{"type": "event", "event": map[string]any{"type": "user/message", "seq": n, "data": map[string]any{"text": "hi"}}},
			},
		})
		_ = wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: open.StreamID, Value: snapRaw})
		if n == 1 {
			// Drop carrier after first snapshot.
			_ = c.Close(websocket.StatusGoingAway, "carrier lost")
			return
		}
		// Second live connection stays open until test cancels.
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()

	proj := Project{ID: "p1", CoordinatorSessionID: "parent-1"}
	var detail WorkDetail
	if err := json.Unmarshal([]byte(`{"workstream":{"id":"w1","projectId":"p1","workerSessionId":"child-1"},"worker":{"sessionId":"child-1","materialized":true}}`), &detail); err != nil {
		t.Fatal(err)
	}

	var liveSnapshots atomic.Int32
	var sawDisconnect atomic.Bool
	err := FollowProcess(ctx, mux, proj, detail, func(s ProcessSnapshot) bool {
		if !s.Live && s.Error != "" {
			sawDisconnect.Store(true)
			return true
		}
		if s.Type == "snapshot" {
			if liveSnapshots.Add(1) >= 2 {
				return false // stop once reconnected snapshot arrives
			}
		}
		return true
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected clean stop after reconnected snapshot, got err=%v snapshots=%d conns=%d", err, liveSnapshots.Load(), connNum.Load())
	}
	if !sawDisconnect.Load() {
		t.Fatal("expected disconnect notification during reconnect backoff")
	}
	if connNum.Load() < 3 {
		t.Fatalf("expected retry past failed 2nd dial to 3rd connection, got %d", connNum.Load())
	}
}
