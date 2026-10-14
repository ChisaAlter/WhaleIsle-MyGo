package schedule

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"whaleisle/internal/dsh"
)

func TestCapturedOwnerUpdateFailureThenSuccess(t *testing.T) {
	record := json.RawMessage(`{"id":"task","title":"Before","prompt":"hello","kind":"daily","time":"09:00:00","timeZone":"Asia/Shanghai","scheduledAt":"2026-12-01T01:00:00Z","sessionId":"owner","status":"active"}`)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args struct {
					Request map[string]json.RawMessage `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		value := any([]any{})
		if strings.HasSuffix(r.URL.Path, "schedule/update") {
			req := body.Payload.Args.Request
			if string(req["sessionId"]) != `"owner"` {
				t.Errorf("wrong target %s", req["sessionId"])
			}
			var expected map[string]json.RawMessage
			_ = json.Unmarshal(req["expected"], &expected)
			if expected["status"] != nil || expected["sessionId"] != nil || string(expected["title"]) != `"Before"` {
				t.Errorf("expected snapshot %s", req["expected"])
			}
			if calls.Add(1) == 1 {
				value = map[string]any{"id": "task", "updated": false, "code": "schedule_conflict"}
			} else {
				value = map[string]any{"id": "task", "updated": true, "record": json.RawMessage(`{"id":"task","title":"After","prompt":"hello","kind":"daily","time":"09:00:00","timeZone":"Asia/Shanghai","scheduledAt":"2026-12-01T01:00:00Z"}`)}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	row := Entry{ID: "task", SessionID: "owner", Raw: record, Origin: srv.URL}
	s.rows = []Entry{row}
	s.sessionID = "different-selected-session"
	title := "After"
	if err := s.Update(context.Background(), row, &title, nil, nil); err == nil || !strings.Contains(err.Error(), "schedule_conflict") {
		t.Fatalf("conflict reported as success: %v", err)
	}
	if string(s.Snapshot().Rows[0].Raw) != string(record) {
		t.Fatal("failure replaced captured record")
	}
	if err := s.Update(context.Background(), row, &title, nil, nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("writes=%d", calls.Load())
	}
}
