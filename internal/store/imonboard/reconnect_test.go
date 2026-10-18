package imonboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestFeishuReconnectTestRequiresConfirmation(t *testing.T) {
	calls := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string         `json:"rpcId"`
			Method  string         `json:"method"`
			Payload map[string]any `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		calls <- req.Method
		value := map[string]any{"bots": []any{map[string]any{"botId": "bot1", "connected": true}}}
		if req.Method == "bot.reconnect" {
			if req.Payload["botId"] != "bot1" || req.Payload["sendTest"] != true {
				t.Errorf("wrong test destination or flag: %#v", req.Payload)
			}
			value["testMessage"] = map[string]any{"sent": true}
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer server.Close()
	s := &Store{client: &pluginrpc.Client{BaseURL: server.URL, HTTPClient: server.Client()}, view: View{Open: true, Available: true, Scope: 3, Channel: "feishu"}}
	s.Review(3, "bot.reconnect-test", "bot1", "", "")
	select {
	case method := <-calls:
		t.Fatal("review sent request", method)
	default:
	}
	s.Confirm()
	select {
	case method := <-calls:
		if method != "bot.reconnect" {
			t.Fatal(method)
		}
	case <-time.After(time.Second):
		t.Fatal("confirmed test reconnect did not dispatch")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if v := s.Snapshot(); !v.Busy && strings.Contains(v.TestNotice, "测试消息已发送") {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("test receipt lost during status refresh: %+v", s.Snapshot())
}
func TestFeishuReconnectReceiptStates(t *testing.T) {
	for _, tc := range []struct {
		body, contains string
		failed         bool
	}{
		{`{"bots":[{"botId":"b","connected":true}],"testMessage":{"sent":false,"code":"test-target-unavailable"}}`, "尚未收到", false},
		{`{"bots":[{"botId":"b","connected":true}],"testMessage":{"sent":false}}`, "发送失败", false},
		{`{"bots":[{"botId":"b","connected":false}],"testMessage":{"sent":true}}`, "仍未连接", true},
	} {
		notice, failure := reconnectTestResult(json.RawMessage(tc.body), "b")
		if tc.failed != (failure != "") || !strings.Contains(notice+failure, tc.contains) {
			t.Fatal(notice, failure)
		}
	}
}
