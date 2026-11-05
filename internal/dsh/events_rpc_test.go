package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The real Host validates the Connection envelope before reading the event
// outcome. A fixture accepting a bare EventResult cannot catch that boundary.
func decodeEventResultRPC(t *testing.T, r *http.Request) (EventResult, string) {
	t.Helper()
	if r.Method != http.MethodPost || r.URL.Path != EventsResultPath || r.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("event RPC request: %s %s %s", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
	}
	var wire map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
		t.Fatal(err)
	}
	if len(wire) != 4 || string(wire["type"]) != `"client-request"` || string(wire["method"]) != `"$events/result"` {
		t.Fatalf("missing exact Connection envelope: %v", wire)
	}
	var rpcID string
	if json.Unmarshal(wire["rpcId"], &rpcID) != nil || rpcID == "" {
		t.Fatal("missing event receipt correlation")
	}
	var payload map[string]json.RawMessage
	if json.Unmarshal(wire["payload"], &payload) != nil || len(payload) != 1 || len(payload["args"]) == 0 {
		t.Fatalf("missing exact payload.args: %s", wire["payload"])
	}
	var result EventResult
	if err := json.Unmarshal(payload["args"], &result); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload["args"]), "Delivery") || strings.Contains(string(payload["args"]), "Context") {
		t.Fatal("event delivery authority entered the wire")
	}
	return result, rpcID
}

func TestPostEventResultRPCEnvelope(t *testing.T) {
	for _, outcome := range []EventResultOutcome{
		{Kind: "next"},
		{Kind: "result", Value: json.RawMessage(`{"answers":[{"id":"choice","selected":["Blue"]}]}`)},
		{Kind: "result", Value: json.RawMessage(`"rejected"`)},
		{Kind: "rejected", Error: &EventRejection{Name: "UserQuestionError", Code: "ASK_CANCELLED", Message: "User declined to answer"}},
	} {
		t.Run(outcome.Kind+string(outcome.Value), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, rpcID := decodeEventResultRPC(t, r)
				if r.Header.Get("Cookie") != "owned=receipt" || got.ClientID != "listener" || got.EventID != "question" {
					t.Fatalf("event authority/correlation changed: %+v", got)
				}
				want, _ := json.Marshal(outcome)
				actual, _ := json.Marshal(got.Outcome)
				if string(want) != string(actual) {
					t.Fatalf("outcome changed: got %s want %s", actual, want)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": rpcID, "result": map[string]any{"ok": true}})
			}))
			defer srv.Close()
			client := &Client{Endpoint: Endpoint{Origin: srv.URL, Cookie: "owned=receipt"}}
			if err := client.PostEventResult(context.Background(), EventResult{ClientID: "listener", EventID: "question", Outcome: outcome}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPostEventResultRPCRejectsBadReceipt(t *testing.T) {
	for _, mode := range []string{"host-error", "wrong-rpc-id", "empty-body"} {
		t.Run(mode, func(t *testing.T) {
			client := &Client{Endpoint: Endpoint{Origin: "https://offline.invalid"}, HTTP: &http.Client{Transport: deliveryRoundTrip(func(r *http.Request) (*http.Response, error) {
				_, rpcID := decodeEventResultRPC(t, r)
				result := map[string]any{"ok": true}
				if mode == "host-error" {
					result = map[string]any{"ok": false, "error": map[string]any{"code": "gateway/bad-request", "message": "rejected receipt", "details": map[string]any{"field": "outcome"}}}
				} else if mode == "wrong-rpc-id" {
					rpcID = "foreign"
				}
				body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": rpcID, "result": result})
				if mode == "empty-body" {
					body = nil
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}, nil
			})}}
			err := client.PostEventResult(context.Background(), EventResult{ClientID: "listener", EventID: "question", Outcome: EventResultOutcome{Kind: "next"}})
			if err == nil {
				t.Fatal("invalid receipt acknowledged")
			}
			if mode == "host-error" {
				var rpcError *Error
				if !errors.As(err, &rpcError) || rpcError.Code != "gateway/bad-request" || rpcError.Message != "rejected receipt" || string(rpcError.Details) != `{"field":"outcome"}` {
					t.Fatalf("Host error details lost: %v", err)
				}
			}
		})
	}
}
