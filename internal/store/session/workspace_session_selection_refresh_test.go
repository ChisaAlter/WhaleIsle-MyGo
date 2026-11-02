package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

// A sidebar-created session needs the same initial projections, skills and
// feedback as one selected through the ordinary session-list action.
func TestWorkspaceSessionSelectionRefreshesSessionViews(t *testing.T) {
	requests := make(chan workspaceCreationCall, 32)
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://workspace-refresh.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					Request map[string]any `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		method := strings.TrimPrefix(r.URL.Path, "/api/")
		requests <- workspaceCreationCall{Method: method, Request: body.Payload.Args.Request}
		value := map[string]any{}
		if method == "session/create" {
			value["sessionId"] = "created"
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	s.Attach(client, nil)
	s.applyWorkspaceBaseline(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"target","path":"/target","title":"Target","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	s.mu.Lock()
	s.selectedID = "source"
	s.mu.Unlock()
	target, err := s.CaptureWorkspaceSessionTarget(s.Snapshot().WorkspaceLayout.HostGeneration, "target")
	if err != nil {
		t.Fatal(err)
	}
	id, err := target.Create(context.Background())
	if err != nil || id != "created" {
		t.Fatalf("create id=%q err=%v", id, err)
	}
	if !target.Select(id) {
		t.Fatal("selection was rejected")
	}
	wanted := map[string]bool{"session/projections": false, "skills/list": false, "messageFeedback/list": false}
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		complete := true
		for _, seen := range wanted {
			complete = complete && seen
		}
		if complete {
			return
		}
		select {
		case call := <-requests:
			if _, ok := wanted[call.Method]; ok {
				if got := call.Request["sessionId"]; got != "created" {
					t.Fatalf("%s refreshed session %v", call.Method, got)
				}
				wanted[call.Method] = true
			}
		case <-deadline.C:
			t.Fatalf("session selection did not refresh views: %v", wanted)
		}
	}
}
