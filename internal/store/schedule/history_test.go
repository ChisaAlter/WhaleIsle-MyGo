package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"whaleisle/internal/dsh"
)

func TestDeliveryHistorySavedFieldsAndDedup(t *testing.T) {
	first, err := DecodeDeliveryPage(json.RawMessage(`{"id":"task","records":[{"messageId":"new","scheduledAt":"2026-10-16T01:00:00Z","deliveredAt":"2026-10-16T01:00:02Z","prompt":"original sent prompt"},{"messageId":"legacy","scheduledAt":"2026-10-15T01:00:00Z","deliveredAt":"2026-10-15T01:00:03Z"}],"nextBefore":"legacy","retention":{"days":30,"records":100},"earlierRecordsUnavailable":true,"earlierRecordsPruned":false}`), "task")
	if err != nil {
		t.Fatal(err)
	}
	older, err := DecodeDeliveryPage(json.RawMessage(`{"id":"task","records":[{"messageId":"legacy","scheduledAt":"2026-10-15T01:00:00Z","deliveredAt":"2026-10-15T01:00:03Z"},{"messageId":"old","scheduledAt":"2026-10-14T01:00:00Z","deliveredAt":"2026-10-14T01:00:02Z","prompt":""},{"messageId":"old","scheduledAt":"2026-10-14T01:00:00Z","deliveredAt":"2026-10-14T01:00:02Z","prompt":""}],"earlierRecordsUnavailable":true,"earlierRecordsPruned":true,"retention":{"days":14,"records":40}}`), "task")
	if err != nil {
		t.Fatal(err)
	}
	merged := MergeDeliveryPages(first, older)
	if len(merged.Records) != 3 || merged.Records[0].MessageID != "new" || merged.Records[2].MessageID != "old" || merged.NextBefore != "" || !merged.EarlierRecordsPruned || merged.Retention.Days != 14 {
		t.Fatalf("bad merge %+v", merged)
	}
	if merged.Records[1].Prompt != nil || merged.Records[2].Prompt == nil || *merged.Records[2].Prompt != "" {
		t.Fatal("legacy absence confused with empty saved prompt")
	}
	if first.NextBefore != "legacy" || len(first.Records) != 2 {
		t.Fatal("previous page mutated")
	}
	for _, raw := range []string{`{"id":"other","records":[]}`, `{"id":"task"}`, `{"id":"task","records":[{"messageId":"x"}]}`} {
		if _, err := DecodeDeliveryPage(json.RawMessage(raw), "task"); err == nil {
			t.Fatal("accepted wrong/incomplete history")
		}
	}
	for _, code := range []string{"schedule_not_found", "delivery_cursor_not_found"} {
		_, err := DecodeDeliveryPage(json.RawMessage(`{"id":"task","code":"`+code+`"}`), "task")
		var result *ResultError
		if !errors.As(err, &result) || result.Code != code {
			t.Fatalf("lost failure code %v", err)
		}
	}
}

func TestDeliveryHistoryCapturedOwnerAndCursor(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args struct {
					Request map[string]json.RawMessage `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		req := body.Payload.Args.Request
		if r.URL.Path != "/api/rpc/schedule/history" && r.URL.Path != "/rpc/schedule/history" {
			t.Logf("Host RPC path: %s", r.URL.Path)
		}
		if string(req["id"]) != `"task"` || string(req["sessionId"]) != `"owner"` || string(req["before"]) != `"opaque"` || string(req["limit"]) != "20" {
			t.Errorf("wrong request %+v", req)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": json.RawMessage(`{"id":"task","records":[],"earlierRecordsPruned":false,"earlierRecordsUnavailable":false,"retention":{"days":30,"records":100}}`)}})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.sessionID = "another session"
	if _, err := s.DeliveryHistory(context.Background(), Entry{ID: "task", SessionID: "owner", Origin: srv.URL}, "opaque"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeliveryHistory(context.Background(), Entry{ID: "task", SessionID: "owner", Origin: "https://other.invalid"}, "opaque"); err == nil {
		t.Fatal("accepted another Host")
	}
	if calls != 1 {
		t.Fatalf("calls %d", calls)
	}
}
