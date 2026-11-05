package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type botTaskTransport func(*http.Request) (*http.Response, error)

func (f botTaskTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	return fixtureResponseIdentity(r, response, err)
}
func botTaskFixture() *DshbotTaskView {
	v := &DshbotTaskView{NS: "dsh-bot", Revision: 8}
	v.Value.Items = []DshbotCatalogItem{{ID: "b", Name: "Test Bot", SessionID: "session-b"}}
	v.Value.Tasks = []DshbotTask{{ID: "t", FromID: "a", ToID: "b", Task: "Synthetic task", Status: "queued"}}
	return v
}
func TestBotTaskRPCContract(t *testing.T) {
	for _, action := range []string{"describe", "pause", "resume", "cancel", "retry", "stop"} {
		t.Run(action, func(t *testing.T) {
			count := 0
			c := &Client{BaseURL: "http://127.0.0.1", Cookie: "synthetic-account", HTTPClient: &http.Client{Transport: botTaskTransport(func(r *http.Request) (*http.Response, error) {
				count++
				var req struct {
					Type, RPCID, Method string
					Payload             map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatal(err)
				}
				method := "task/" + action
				if action == "describe" {
					method = "describe"
				}
				if action == "stop" {
					method = "bot/stop"
				}
				if r.Method != "POST" || r.URL.Path != "/dshbot/"+method || req.Method != method || req.Type != "client-request" || r.Header.Get("Cookie") != "synthetic-account" {
					t.Fatalf("wrong request: %+v %s", req, r.URL)
				}
				result := DshbotTaskResult{View: botTaskFixture(), TaskID: "t", StopAccepted: true, WakeAccepted: true}
				if action == "describe" {
					if req.Payload["botId"] != "b" || len(req.Payload) != 1 {
						t.Fatalf("describe payload: %+v", req.Payload)
					}
				} else {
					if req.Payload["revision"] != float64(7) {
						t.Fatalf("missing revision: %+v", req.Payload)
					}
					if action == "stop" {
						if req.Payload["botId"] != "b" || len(req.Payload) != 2 {
							t.Fatal(req.Payload)
						}
					} else {
						if req.Payload["taskId"] != "t" {
							t.Fatal(req.Payload)
						}
					}
				}
				if action == "retry" {
					if req.Payload["requestId"] != "same-operation" || len(req.Payload) != 3 {
						t.Fatal(req.Payload)
					}
					result.TaskID = "replacement"
					result.OriginalTaskID = "t"
					result.View.Value.Tasks = append(result.View.Value.Tasks, DshbotTask{ID: "replacement", FromID: "a", ToID: "b", ParentTaskID: "t", RetryRequestID: "same-operation", Status: "queued"})
				}
				raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": result}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
			})}}
			if action == "describe" {
				if _, err := c.DshbotTasks(context.Background(), "b"); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := c.DshbotControlTask(context.Background(), DshbotTaskCommand{Action: action, BotID: "b", TaskID: "t", Revision: 7, RequestID: "same-operation"}); err != nil {
					t.Fatal(err)
				}
			}
			if count != 1 {
				t.Fatalf("unexpected retries: %d", count)
			}
		})
	}
}
func TestBotTaskRejectsInvalidAndMismatchedReceipts(t *testing.T) {
	calls := 0
	c := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: botTaskTransport(func(*http.Request) (*http.Response, error) {
		calls++
		v := botTaskFixture()
		v.Value.Tasks[0].ToID = "other"
		raw, _ := json.Marshal(map[string]any{"result": map[string]any{"ok": true, "value": DshbotTaskResult{View: v, TaskID: "t"}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})}}
	for _, cmd := range []DshbotTaskCommand{{Action: "invalid", BotID: "b"}, {Action: "retry", BotID: "b", TaskID: "t"}, {Action: "pause", BotID: "b", Revision: -1, TaskID: "t"}} {
		if _, err := c.DshbotControlTask(context.Background(), cmd); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
	if calls != 0 {
		t.Fatal("invalid input reached transport")
	}
	if _, err := c.DshbotControlTask(context.Background(), DshbotTaskCommand{Action: "pause", BotID: "b", TaskID: "t", Revision: 7}); err == nil {
		t.Fatal("foreign task receipt accepted")
	}
	if calls != 1 {
		t.Fatal("transport automatically retried")
	}
}
func TestBotTaskActionStatusContract(t *testing.T) {
	allowed := map[string]string{"pause": "queued", "resume": "paused", "cancel": "queued paused delivered", "retry": "failed cancelled"}
	for action, statuses := range allowed {
		for _, status := range []string{"queued", "paused", "delivered", "failed", "cancelled", "completed", "expired", "running"} {
			want := strings.Contains(" "+statuses+" ", " "+status+" ")
			if DshbotTaskActionAllowed(action, status) != want {
				t.Fatalf("%s %s", action, status)
			}
		}
	}
}
