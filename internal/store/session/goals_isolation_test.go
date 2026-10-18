package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func TestGoalLateResultsAreSessionAndGenerationScoped(t *testing.T) {
	for _, operation := range []string{"get", "create", "edit", "complete", "pause", "resume"} {
		for _, selection := range []string{"A-B", "A-B-A"} {
			for _, failure := range []bool{false, true} {
				t.Run(operation+"/"+selection+map[bool]string{false: "/success", true: "/error"}[failure], func(t *testing.T) {
					started := make(chan struct{})
					release := make(chan struct{})
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var body map[string]any
						_ = json.NewDecoder(r.Body).Decode(&body)
						if r.URL.Path != "/api/goals/"+operation {
							t.Errorf("unexpected RPC %s", r.URL.Path)
						}
						close(started)
						<-release
						if failure {
							http.Error(w, "old A failed", http.StatusBadGateway)
							return
						}
						value := map[string]any{"id": "old-A", "revision": 2, "objective": "old A goal", "phase": "paused", "ref": map[string]any{"id": "old-A", "revision": 2}}
						rpcOK(w, body["rpcId"], value)
					}))
					defer srv.Close()
					s := New()
					client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
					s.Attach(client, nil)
					s.selectedID = "A"
					generation := s.followGeneration
					finished := make(chan struct{})
					go func() {
						defer close(finished)
						ref := &api.GoalRef{Id: "old-A", Revision: 1}
						switch operation {
						case "get":
							s.runRefreshGoal(context.Background(), client, "A", generation)
						case "create":
							s.runCreateGoal(context.Background(), client, "A", generation, "old A goal")
						case "edit":
							s.runEditGoal(context.Background(), client, "A", generation, "old A goal", ref)
						default:
							s.runGoalAction(context.Background(), client, "A", generation, "goals/"+operation, "old A action", operation == "complete", ref)
						}
					}()
					wait(t, started)
					s.mu.Lock()
					s.selectSessionLocked("B")
					if selection == "A-B-A" {
						s.selectSessionLocked("A")
					}
					s.planBoard = PlanBoard{Goal: &PlanGoal{ID: "new", Objective: "new goal"}, Todos: []PlanTodo{{Content: "new todo"}}}
					s.lastAction = "new status"
					s.mu.Unlock()
					before := s.Snapshot()
					close(release)
					wait(t, finished)
					if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
						t.Fatalf("old goal result mutated current UI: %+v", got)
					}
				})
			}
		}
	}
}
