package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
)

func postMergeRPCReply(t *testing.T, request *http.Request, value any) *http.Response {
	t.Helper()
	var body struct {
		RPCID string `json:"rpcId"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
	if err != nil {
		t.Fatal(err)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}
}

func TestGoalDockClearRetiresEarlierRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://goal-owner.invalid"}, HTTP: &http.Client{Transport: goalReadRoundTrip(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/api/goals/get":
				close(started)
				<-release
				return postMergeRPCReply(t, r, map[string]any{"id": "goal", "revision": 3, "objective": "already cleared", "phase": "active"}), nil
			case "/api/goals/clear":
				return postMergeRPCReply(t, r, nil), nil
			default:
				t.Fatalf("unexpected RPC: %s", r.URL.Path)
				return nil, nil
			}
		})}}
		s := New()
		s.Attach(client, nil)
		s.selectedID = "A"
		s.planBoard.Goal = &PlanGoal{ID: "goal", Revision: 3}
		s.RefreshGoal(context.Background())
		<-started
		if err := s.GoalDockAction("goal", 3)(context.Background(), "clear", ""); err != nil {
			t.Fatal(err)
		}
		close(release)
		synctest.Wait()
		if got := s.Snapshot().PlanBoard.Goal; got != nil {
			t.Fatalf("old read resurrected cleared goal: %+v", got)
		}
	})
}

func TestLegacyPromptCompletionKeepsItsHostAndNavigation(t *testing.T) {
	for _, change := range []string{"replace-host", "navigate-away-and-back"} {
		t.Run(change, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				started, release := make(chan struct{}), make(chan struct{})
				client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://prompt-owner.invalid"}, HTTP: &http.Client{Transport: goalReadRoundTrip(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path != "/api/session/prompt" {
						t.Fatalf("unexpected RPC: %s", r.URL.Path)
					}
					close(started)
					<-release
					return postMergeRPCReply(t, r, map[string]any{"accepted": true}), nil
				})}}
				s := New()
				s.Attach(client, nil)
				s.selectedID = "A"
				s.SendPrompt(context.Background(), "old private draft")
				<-started
				if change == "replace-host" {
					s.Attach(&dsh.Client{}, nil)
				} else {
					s.mu.Lock()
					s.selectSessionLocked("B")
					s.selectSessionLocked("A")
					s.mu.Unlock()
				}
				s.mu.Lock()
				s.messages = []Message{{ID: "current", Role: "assistant", Body: "current owner"}}
				s.lastAction = "current action"
				s.mu.Unlock()
				close(release)
				synctest.Wait()
				got := s.Snapshot()
				if len(got.Messages) != 1 || got.Messages[0].Body != "current owner" || got.LastAction != "current action" {
					t.Fatalf("late prompt changed replacement view: messages=%+v action=%q", got.Messages, got.LastAction)
				}
			})
		})
	}
}
