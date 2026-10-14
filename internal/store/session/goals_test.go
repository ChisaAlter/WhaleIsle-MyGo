package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestCreateGoalAccepted(t *testing.T) {
	var gotAgent, gotObjective string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/goals/create") {
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			gotAgent, _ = args["agentId"].(string)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				gotObjective, _ = req["objective"].(string)
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"ref": map[string]any{"id": "g1", "revision": 1.0},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	done := make(chan struct{})
	st.Attach(client, func() {
		if st.Snapshot().LastAction == "goal created" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.Select("s1")
	st.CreateGoal(context.Background(), " ship it ")
	wait(t, done)
	if gotAgent != "s1" || gotObjective != "ship it" {
		t.Fatalf("agent/objective = %q/%q", gotAgent, gotObjective)
	}
	g := st.Snapshot().PlanBoard.Goal
	if g == nil || g.ID != "g1" || g.Objective != "ship it" {
		t.Fatalf("goal = %#v", g)
	}
}

func TestPauseAndClearGoal(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/goals/pause"):
			methods = append(methods, "pause")
			rpcOK(w, body["rpcId"], map[string]any{
				"id": "g1", "revision": 2.0, "objective": "x", "phase": "paused",
			})
		case strings.HasSuffix(path, "/goals/clear"):
			methods = append(methods, "clear")
			rpcOK(w, body["rpcId"], map[string]any{"id": "g1", "revision": 2.0})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	paused := make(chan struct{})
	cleared := make(chan struct{})
	st.Attach(client, func() {
		switch st.Snapshot().LastAction {
		case "goal paused":
			select {
			case <-paused:
			default:
				close(paused)
			}
		case "goal cleared":
			select {
			case <-cleared:
			default:
				close(cleared)
			}
		}
	})
	st.Select("s1")
	st.mu.Lock()
	st.planBoard.Goal = &PlanGoal{ID: "g1", Revision: 1, Objective: "x", Phase: "active"}
	st.mu.Unlock()

	st.PauseGoal(context.Background())
	wait(t, paused)
	if g := st.Snapshot().PlanBoard.Goal; g == nil || g.Phase != "paused" {
		t.Fatalf("after pause: %#v", g)
	}
	st.ClearGoal(context.Background())
	wait(t, cleared)
	if st.Snapshot().PlanBoard.Goal != nil {
		t.Fatal("expected nil goal after clear")
	}
	if len(methods) != 2 || methods[0] != "pause" || methods[1] != "clear" {
		t.Fatalf("methods=%v", methods)
	}
}
