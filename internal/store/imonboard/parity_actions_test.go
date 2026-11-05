package imonboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestParityProvisionErrorAndTargetedRefresh(t *testing.T) {
	if message := publicOperationError(fmt.Errorf("TOKEN_REFUSED credential=fixture-secret"), map[string]any{"appSecret": "fixture-secret"}); !strings.Contains(message, "TOKEN_REFUSED") || strings.Contains(message, "fixture-secret") {
		t.Fatal("public platform reason lost or credential echoed", message)
	}
	failed := decodeProvision(json.RawMessage(`{"attemptId":"a","status":"failed","error":{"code":"denied","message":"admin denied"}}`), Provision{})
	if failed.Error.Code != "denied" || failed.Error.Message != "admin denied" {
		t.Fatal("platform failure details discarded", failed)
	}
	for _, op := range []struct{ operation, method string }{{"callback_repair", "bot.callback-repair.begin"}, {"group_message_permission", "bot.group-message-permission.begin"}} {
		t.Run(op.operation, func(t *testing.T) {
			calls := make(chan string, 8)
			transport := reconnectFixtureTransport(func(r *http.Request) (*http.Response, error) {
				var req struct {
					RPCID, Method string
					Payload       map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					return nil, err
				}
				calls <- req.Method
				value := map[string]any{"status": "cancelled"}
				if req.Method == op.method {
					if req.Payload["botId"] != "original-bot" || req.Payload["locale"] != nil {
						t.Error("targeted refresh lost Bot or switched to new-registration payload", req.Payload)
					}
					value = map[string]any{"attemptId": "new", "status": "cancelled", "operation": op.operation, "botId": "original-bot"}
				}
				body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
			})
			s := Store{client: &pluginrpc.Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: transport}}, view: View{Open: true, Available: true, Scope: 1, Channel: "feishu", Provision: Provision{AttemptID: "old", Operation: op.operation, BotID: "original-bot"}}}
			s.Review(1, "provision.begin", "", "", "")
			s.Confirm()
			for _, want := range []string{"provision.cancel", op.method} {
				select {
				case got := <-calls:
					if got != want {
						t.Fatalf("%s want %s", got, want)
					}
				case <-time.After(2 * time.Second):
					t.Fatal("targeted refresh did not execute")
				}
			}
			s.Hide()
		})
	}
}
