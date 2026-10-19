package projectflow

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type requestQueueTransport func(*http.Request) (*http.Response, error)

func (f requestQueueTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestQueueUsesSourceProjectionAndRevisionContract(t *testing.T) {
	changes := 0
	available := true
	client := Client{Origin: "http://127.0.0.1:1", HTTP: &http.Client{Transport: requestQueueTransport(func(r *http.Request) (*http.Response, error) {
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			t.Fatal(err)
		}
		if r.URL.Path != "/dsh-project/"+q.Method || q.Payload["projectId"] != "project" {
			t.Fatal("wrong queue route/project", q)
		}
		var value any
		switch q.Method {
		case "detail":
			project := map[string]any{"id": "project"}
			if available {
				project["queue"] = map[string]any{"active": map[string]any{"id": "request", "revision": 3}, "queued": []any{map[string]any{"id": "queued", "revision": 2, "position": 1}}, "paused": true}
			}
			value = map[string]any{"project": project}
		case "requests/queue":
			changes++
			if q.Payload["requestId"] != "request" || q.Payload["requestRevision"] != float64(3) || q.Payload["action"] != "move" || q.Payload["beforeRequestId"] != "queued" {
				t.Fatalf("changed source payload: %#v", q.Payload)
			}
			value = map[string]any{"requestId": "request", "queue": map[string]any{"state": "queued", "position": 1}, "projectQueue": map[string]any{"queued": []any{map[string]any{"id": "request", "revision": 3}}}}
		default:
			t.Fatal("invented queue endpoint", q.Method)
		}
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	queue, err := client.RequestQueue(context.Background(), "project")
	if err != nil || queue.Active.ID != "request" || len(queue.Queued) != 1 || !queue.Paused {
		t.Fatal(queue, err)
	}
	result, err := client.ChangeRequestQueue(context.Background(), "project", RequestQueueChange{RequestID: "request", RequestRevision: 3, Action: "move", BeforeRequestID: "queued"})
	if err != nil || result.Queue.Position != 1 || result.ProjectQueue.Queued[0].ID != "request" {
		t.Fatal(result, err)
	}
	for _, bad := range []RequestQueueChange{{RequestID: "request", Action: "cancel"}, {RequestID: "request", RequestRevision: 3, Action: "moveUp"}, {RequestID: "request", RequestRevision: 3, Action: "cancel", BeforeRequestID: "queued"}} {
		if _, err := client.ChangeRequestQueue(context.Background(), "project", bad); err == nil {
			t.Fatal("invalid queue contract was sent")
		}
	}
	if changes != 1 {
		t.Fatal("invalid queue action mutated fixture")
	}
	available = false
	if _, err := client.RequestQueue(context.Background(), "project"); err == nil {
		t.Fatal("missing capability silently became an empty queue")
	}
}
