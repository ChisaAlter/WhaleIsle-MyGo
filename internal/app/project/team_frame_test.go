package projectctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

// The first frame of an ordinary session's Team view must never report
// Loading=false with a nil Team. That flash used to be reachable because the
// model was assembled first and the first read started afterwards: a read that
// finished in between cleared the flag the model re-read while the Team it
// snapshotted stayed nil.
//
// This pins the ordering rather than the timing. The read asks its dependencies
// for a context only after the state is marked loading, so the flag a frame
// snapshots can never be cleared by a read that had not been claimed yet.
func TestOrdinaryTeamFirstFrameClaimsReadBeforeAskingDependencies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		value := map[string]any{
			"items": []any{map[string]any{"sessionId": "s1", "running": false}},
		}
		if r.URL.Path != "/api/session/list" {
			value = map[string]any{
				"values": map[string]any{
					"agentTeam": map[string]any{
						"members": []any{map[string]any{"id": "s1", "name": "lead", "role": "lead", "phase": "active"}},
						"tasks":   []any{},
					},
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  request.RPCID,
			"result": map[string]any{"ok": true, "value": value},
		})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: srv.Client()}
	store := session.New()
	store.Select("s1")

	var asked, loadingWhenAsked atomic.Bool
	var mgr *Manager
	askedWithoutLoading := func() {
		asked.Store(true)
		mgr.mu.Lock()
		loading := mgr.ordinaryTeam != nil && mgr.ordinaryTeam.loading
		mgr.mu.Unlock()
		if loading {
			loadingWhenAsked.Store(true)
		}
	}
	mgr = NewManager(HostDeps{
		HostSnapshot:     func() HostSnapshot { return HostSnapshot{Client: client} },
		Sessions:         func() *session.Store { return store },
		LifecycleContext: func() context.Context { askedWithoutLoading(); return context.Background() },
	})

	first := mgr.OrdinaryTeamModel(nil)
	if !first.Loading && first.Team == nil {
		t.Fatal("first frame of OrdinaryTeamModel returned Loading=false && Team==nil")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && mgr.OrdinaryTeamModel(nil).Team == nil {
		time.Sleep(5 * time.Millisecond)
	}
	if !asked.Load() {
		t.Fatal("the first frame did not start a read")
	}
	if !loadingWhenAsked.Load() {
		t.Fatal("the first read asked for its dependencies before the state was marked loading, so a read that finished early could clear a flag a frame had already snapshotted")
	}
}
