package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

type goalReadRoundTrip func(*http.Request) (*http.Response, error)

func (f goalReadRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Exercise the real Client.Call envelope/decoder without opening a connection.
func goalReadClient(t *testing.T, result map[string]any, beforeReply func()) *dsh.Client {
	t.Helper()
	return &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://goal-fixture.invalid"}, HTTP: &http.Client{Transport: goalReadRoundTrip(func(r *http.Request) (*http.Response, error) {
		var request struct {
			RPCID   string `json:"rpcId"`
			Method  string `json:"method"`
			Payload struct {
				Args struct {
					AgentID string `json:"agentId"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, err
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/goals/get" || request.Method != "goals/get" || request.Payload.Args.AgentID != "A" {
			t.Errorf("unexpected background goal RPC %s %s method=%s agent=%s", r.Method, r.URL.Path, request.Method, request.Payload.Args.AgentID)
		}
		if beforeReply != nil {
			beforeReply()
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": result})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
}

func TestGoalReadOptionalSuccessDoesNotPolluteAction(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result map[string]any
		goal   bool
	}{
		{"undefined", map[string]any{"ok": true}, false},
		{"null", map[string]any{"ok": true, "value": nil}, false},
		{"empty", map[string]any{"ok": true, "value": map[string]any{}}, false},
		{"goal", map[string]any{"ok": true, "value": map[string]any{"id": "current", "objective": "fixture objective", "revision": 2, "phase": "active"}}, true},
	} {
		for _, status := range []string{"", "prompt failed: fixture denied"} {
			t.Run(tc.name+"/"+status, func(t *testing.T) {
				client := goalReadClient(t, tc.result, nil)
				s := New()
				s.Attach(client, nil)
				s.selectedID = "A"
				s.lastAction = status
				s.planBoard.Goal = &PlanGoal{ID: "stale"}
				s.goalReadError = goalReadErrorState{client: client, sessionID: "A", generation: s.followGeneration, message: "previous read failed"}
				s.runRefreshGoal(context.Background(), client, "A", s.followGeneration)
				v := s.Snapshot()
				if v.LastAction != status || v.GoalReadError != "" {
					t.Fatalf("background success changed action or kept old error: action=%q read=%q", v.LastAction, v.GoalReadError)
				}
				if tc.goal {
					if v.PlanBoard.Goal == nil || v.PlanBoard.Goal.ID != "current" || v.PlanBoard.Goal.Revision != 2 {
						t.Fatal("goal result not folded", v.PlanBoard.Goal)
					}
				} else if !v.PlanBoard.Empty() {
					t.Fatal("optional absent goal opened a plan panel", v.PlanBoard)
				}
			})
		}
	}
}

func TestGoalReadRealErrorHasDedicatedOwnership(t *testing.T) {
	result := map[string]any{"ok": false, "error": map[string]any{"code": "GOAL_AGENT_UNAVAILABLE", "message": "fixture unavailable"}}
	client := goalReadClient(t, result, nil)
	s := New()
	s.Attach(client, nil)
	s.selectedID = "A"
	s.lastAction = "prompt failed: fixture denied"
	s.planBoard.Goal = &PlanGoal{ID: "known", Objective: "keep current projection"}
	s.runRefreshGoal(context.Background(), client, "A", s.followGeneration)
	v := s.Snapshot()
	if v.LastAction != "prompt failed: fixture denied" || v.GoalReadError != "goals/get: GOAL_AGENT_UNAVAILABLE: fixture unavailable" || v.PlanBoard.Goal == nil || v.PlanBoard.Goal.ID != "known" {
		t.Fatal("read error lost ownership or goal projection", v.GoalReadError, v.LastAction, v.PlanBoard.Goal)
	}
	// A subsequent legitimate absent result clears only the goal diagnostic.
	result["ok"] = true
	delete(result, "error")
	s.runRefreshGoal(context.Background(), client, "A", s.followGeneration)
	v = s.Snapshot()
	if v.GoalReadError != "" || v.LastAction != "prompt failed: fixture denied" || v.PlanBoard.Goal != nil {
		t.Fatal("optional success did not clear only its own read state")
	}
}

func TestGoalReadErrorProjectionExpiresWithOwner(t *testing.T) {
	for _, change := range []string{"A-B", "A-B-A", "follow-generation", "host-replacement", "host-original-client"} {
		t.Run(change, func(t *testing.T) {
			client := goalReadClient(t, map[string]any{"ok": false, "error": map[string]any{"message": "fixture read failed"}}, nil)
			s := New()
			s.Attach(client, nil)
			s.selectedID = "A"
			s.runRefreshGoal(context.Background(), client, "A", s.followGeneration)
			if s.Snapshot().GoalReadError == "" {
				t.Fatal("fixture did not retain real read error")
			}
			switch change {
			case "A-B", "A-B-A":
				s.mu.Lock()
				s.selectSessionLocked("B")
				if change == "A-B-A" {
					s.selectSessionLocked("A")
				}
				s.mu.Unlock()
			case "follow-generation":
				s.mu.Lock()
				s.invalidateFollowLocked()
				s.mu.Unlock()
			default:
				s.Attach(&dsh.Client{}, nil)
				if change == "host-original-client" {
					s.Attach(client, nil)
				}
			}
			if s.Snapshot().GoalReadError != "" {
				t.Fatal("new owner inherited old read diagnostic")
			}
		})
	}
}

func TestGoalReadLateErrorCannotPolluteReplacement(t *testing.T) {
	for _, change := range []string{"A-B-A", "host-replacement", "host-original-client"} {
		t.Run(change, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			client := goalReadClient(t, map[string]any{"ok": false, "error": map[string]any{"message": "old fixture failure"}}, func() { close(started); <-release })
			s := New()
			s.Attach(client, nil)
			s.selectedID = "A"
			generation := s.followGeneration
			done := make(chan struct{})
			go func() { defer close(done); s.runRefreshGoal(context.Background(), client, "A", generation) }()
			wait(t, started)
			if change == "A-B-A" {
				s.mu.Lock()
				s.selectSessionLocked("B")
				s.selectSessionLocked("A")
				s.mu.Unlock()
			} else {
				s.Attach(&dsh.Client{}, nil)
				if change == "host-original-client" {
					s.Attach(client, nil)
				}
			}
			s.mu.Lock()
			s.lastAction = "new owner"
			s.planBoard.Goal = &PlanGoal{ID: "new-goal"}
			s.mu.Unlock()
			before := s.Snapshot()
			close(release)
			wait(t, done)
			if got := s.Snapshot(); !reflect.DeepEqual(got, before) {
				t.Fatal("late old read failure changed current view", got.GoalReadError, got.LastAction)
			}
		})
	}
}

func TestGoalReadDoesNotChangeWriteErrorRouting(t *testing.T) {
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://goal-fixture.invalid"}, HTTP: &http.Client{Transport: goalReadRoundTrip(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		if r.URL.Path != "/api/goals/create" {
			t.Errorf("unexpected write RPC %s", r.URL.Path)
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": false, "error": map[string]any{"code": "GOAL_WRITE_DENIED", "message": "fixture write denied"}}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	s := New()
	s.Attach(client, nil)
	s.selectedID = "A"
	s.goalReadError = goalReadErrorState{client: client, sessionID: "A", generation: s.followGeneration, message: "goals/get: fixture read failed"}
	s.runCreateGoal(context.Background(), client, "A", s.followGeneration, "fixture objective")
	v := s.Snapshot()
	if v.LastAction != "GOAL_WRITE_DENIED: fixture write denied" || v.GoalReadError != "goals/get: fixture read failed" {
		t.Fatal("write error was hidden or read diagnostic lost", v.LastAction, v.GoalReadError)
	}
}
