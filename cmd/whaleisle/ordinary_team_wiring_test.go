package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
)

func TestOrdinaryTeamModelInitialLoadingAndProjectControllerPreservation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string
			Payload map[string]any
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/api/session/list":
			value = map[string]any{"items": []any{map[string]any{"sessionId": "s1", "running": false}}}
		case "/api/session/projections":
			value = map[string]any{
				"values": map[string]any{
					"agentTeam": map[string]any{
						"members": []any{map[string]any{"id": "s1", "name": "lead", "role": "lead", "phase": "active"}},
						"tasks":   []any{},
					},
				},
			}
		default:
			value = map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  req.RPCID,
			"result": map[string]any{"ok": true, "value": value},
		})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: srv.Client()}
	store := session.New()
	store.Select("s1")

	s := &mainState{sessions: store, dshClient: client}

	// First call to ordinaryTeamModel must immediately reflect Loading=true (no 1-frame flash of "Team 暂不可用").
	m := s.ordinaryTeamModel(nil)
	if !m.Loading && m.Team == nil {
		t.Fatal("first frame of ordinaryTeamModel returned Loading=false && Team==nil (would flash Team 暂不可用)")
	}

	// Calling bindProjectModel when projectFlow.client transitions to s.dshClient must preserve ordinaryTeam.
	parent := frame.Model{ThemeID: "system"}
	s.bindProjectModel(nil, &parent)
	s.mu.Lock()
	preserved := s.projectFlow.ordinaryTeam != nil
	s.mu.Unlock()
	if !preserved {
		t.Fatal("projectController.projectModel wiped out projectFlow.ordinaryTeam")
	}

	// Wait for the async projection read to complete.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m = s.ordinaryTeamModel(nil)
		if m.Team != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if m.Team == nil || len(m.Team.Members) != 1 || m.Team.Members[0].Name != "lead" {
		t.Fatalf("expected loaded Team with member 'lead', got %+v (err=%q)", m.Team, m.Error)
	}
}
