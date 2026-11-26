package schedule

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestUpdateRejectsNullCommittedRecordWithoutPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  body["rpcId"],
			"result": map[string]any{"ok": true, "value": map[string]any{"record": nil}},
		})
	}))
	defer srv.Close()

	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	raw := json.RawMessage(`{"id":"job-1","sessionId":"s1","title":"old","prompt":"run","status":"active","timing":{"kind":"cron","cron":"0 9 * * *"}}`)
	rows := parseScheduleList([]json.RawMessage{raw})
	rows[0].Origin = srv.URL
	s.mu.Lock()
	s.rows = rows
	s.publishLocked()
	s.mu.Unlock()

	newTitle := "updated"
	err := s.Update(context.Background(), rows[0], &newTitle, nil, nil)
	if err == nil {
		t.Fatal("expected error when schedule/edit returns {\"record\": null}")
	}

	// Direct nil-guard verification for mergeCommittedRows on null/non-object JSON.
	rule, _ := RuleSnapshot(rows[0].Raw)
	merged := mergeCommittedRows(rows, rows[0], rule, json.RawMessage(`null`))
	if len(merged) != 1 || merged[0].ID != "job-1" {
		t.Fatalf("unexpected rows after null committed merge: %+v", merged)
	}
}
