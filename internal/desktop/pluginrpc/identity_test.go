package pluginrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallRejectsWrongResponseIdentityBeforeResult(t *testing.T) {
	for _, mode := range []string{"type", "rpcId"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var req clientRequest
				json.NewDecoder(r.Body).Decode(&req)
				kind, id := "server-response", req.RPCID
				if mode == "type" {
					kind = "client-request"
				} else {
					id = "another-request"
				}
				json.NewEncoder(w).Encode(map[string]any{"type": kind, "rpcId": id, "result": map[string]any{"ok": true, "value": map[string]any{"mustNotBeConsumed": true}}})
			}))
			defer srv.Close()
			var out map[string]any
			err := (&Client{BaseURL: srv.URL}).Call(context.Background(), "dsh-whale", "catalog", map[string]any{}, &out)
			if err == nil || !strings.Contains(err.Error(), "identity") || out != nil || calls != 1 {
				t.Fatal("mismatched envelope consumed", err, out, calls)
			}
		})
	}
}

func TestCallMalformedResultIsNotAHostRejection(t *testing.T) {
	for _, result := range []string{
		`null`,
		`{}`,
		`{"error":{"code":"invalid","message":"not accepted"}}`,
		`{"ok":null,"error":{"code":"invalid","message":"not accepted"}}`,
		`{"ok":false,"error":{}}`,
		`{"ok":false,"error":{"code":"invalid"}}`,
		`{"ok":false,"error":{"code":null,"message":"not accepted"}}`,
		`{"ok":false,"error":{"code":"invalid","message":null}}`,
	} {
		t.Run(result, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var req clientRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				_ = json.NewEncoder(w).Encode(serverResponse{Type: "server-response", RPCID: req.RPCID, Result: json.RawMessage(result)})
			}))
			defer srv.Close()
			out := map[string]any{"original": true}
			err := (&Client{BaseURL: srv.URL}).Call(context.Background(), "office", "connector.configure", map[string]any{}, &out)
			var rejected *RPCError
			if err == nil || errors.As(err, &rejected) || calls != 1 || len(out) != 1 || out["original"] != true {
				t.Fatalf("malformed result lost uncertain outcome: err=%v, calls=%d, out=%v", err, calls, out)
			}
		})
	}
}

func TestCallValidFailurePreservesHostError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		_ = json.NewEncoder(w).Encode(serverResponse{Type: "server-response", RPCID: req.RPCID,
			Result: json.RawMessage(`{"ok":false,"error":{"code":"invalid","message":"correct the input","details":{}}}`)})
	}))
	defer srv.Close()
	err := (&Client{BaseURL: srv.URL}).Call(context.Background(), "office", "connector.configure", map[string]any{}, nil)
	var rejected *RPCError
	if !errors.As(err, &rejected) || rejected.Code != "invalid" || rejected.Message != "correct the input" {
		t.Fatalf("valid failure was not preserved: %v", err)
	}
}
