package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

// Mock only: no host process or session log files are opened.
func TestRepairNativeFlow(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{}, 3)
	release := make(chan struct{}, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string         `json:"rpcId"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if r.URL.Path == "/usage-stats/overview" {
			writeRPC(w, req.RPCID, json.RawMessage(statsFixture))
			return
		}
		n := calls.Add(1)
		if r.URL.Path != "/usage-stats/repair.session" || r.Header.Get("Cookie") != "fixture=yes" {
			t.Error("wrong repair endpoint/auth")
		}
		if len(req.Payload) != 1 || req.Payload["sessionId"] != "bad" {
			t.Error("wrong payload")
		}
		started <- struct{}{}
		<-release
		if n == 1 {
			http.Error(w, "mock failure", http.StatusInternalServerError)
			return
		}
		writeRPC(w, req.RPCID, json.RawMessage(`{"repaired":8,"backup":"mock/session.bak-123","bytesBefore":100,"bytesAfter":90}`))
	}))
	defer server.Close()
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL, Cookie: "fixture=yes"}}
	s.Attach(client, nil)
	s.stats.Raw = json.RawMessage(statsFixture)
	wait := func() RepairView {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v := s.Repair()
			if !v.Busy {
				return v
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("repair timed out")
		return RepairView{}
	}
	s.ConfirmRepair(context.Background(), "bad")
	s.PrepareRepair("unknown")
	if s.Repair().PendingID != "" || calls.Load() != 0 {
		t.Fatal("unconfirmed/unknown repair")
	}
	s.PrepareRepair("bad")
	s.CancelRepair()
	s.ConfirmRepair(context.Background(), "bad")
	if calls.Load() != 0 {
		t.Fatal("cancel sent a request")
	}
	s.PrepareRepair("bad")
	s.RefreshStats(context.Background(), true)
	waitStats(t, s)
	if s.Repair().PendingID != "" {
		t.Fatal("refresh retained stale confirmation")
	}
	s.PrepareRepair("bad")
	s.ConfirmRepair(context.Background(), "other")
	if s.Repair().PendingID != "bad" || calls.Load() != 0 {
		t.Fatal("wrong ID confirmed")
	}
	s.ConfirmRepair(context.Background(), "bad")
	<-started
	s.PrepareRepair("bad")
	s.ConfirmRepair(context.Background(), "bad")
	s.RefreshStats(context.Background(), true)
	if calls.Load() != 1 || s.Stats().Loading {
		t.Fatal("duplicate repair/rescan while busy")
	}
	release <- struct{}{}
	if v := wait(); v.Err == "" || v.Result != nil || s.Stats().Overview.AllTime.Totals.Total != 12 {
		t.Fatalf("failure lost prior stats: %+v", v)
	}
	s.PrepareRepair("bad")
	s.ConfirmRepair(context.Background(), "bad")
	<-started
	release <- struct{}{}
	if v := wait(); v.Err != "" || v.Result == nil || v.Result.Repaired != 8 || v.Result.Backup != "mock/session.bak-123" || s.CanRepair("bad") {
		t.Fatalf("bad success: %+v", v)
	}
	s.PrepareRepair("bad")
	s.ConfirmRepair(context.Background(), "bad")
	if calls.Load() != 2 {
		t.Fatal("repaired stale candidate was repeated")
	}
	// A fresh attachment resets candidates/results; an in-flight old response
	// must not leak into that new attachment.
	s.Attach(client, nil)
	s.stats.Raw = json.RawMessage(statsFixture)
	s.PrepareRepair("bad")
	s.ConfirmRepair(context.Background(), "bad")
	<-started
	s.Detach()
	s.Attach(client, nil)
	release <- struct{}{}
	time.Sleep(20 * time.Millisecond)
	if v := s.Repair(); v.Busy || v.Result != nil || v.Err != "" || v.PendingID != "" {
		t.Fatalf("late result leaked: %+v", v)
	}
}
