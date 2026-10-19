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

type reconnectFixtureTransport func(*http.Request) (*http.Response, error)

func (f reconnectFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSupportedChannelReconnectReviewPayloadAndReceipt(t *testing.T) {
	// Every request terminates in this in-memory transport. No socket, listener,
	// credential, platform account, destination or real message exists here.
	for _, platform := range Platforms {
		for _, action := range []struct {
			name     string
			sendTest bool
		}{{"bot.reconnect-test", true}, {"bot.reconnect", false}} {
			t.Run(platform.ID+"/"+action.name, func(t *testing.T) {
				type request struct {
					RPCID   string         `json:"rpcId"`
					Method  string         `json:"method"`
					Payload map[string]any `json:"payload"`
				}
				type captured struct {
					path    string
					request request
				}
				calls := make(chan captured, 4)
				notifications := make(chan struct{}, 16)
				transport := reconnectFixtureTransport(func(r *http.Request) (*http.Response, error) {
					var req request
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						return nil, err
					}
					calls <- captured{path: r.URL.Path, request: req}
					value := map[string]any{"bots": []any{map[string]any{"botId": "fixture-bot", "connected": true}}}
					if req.Method == "bot.reconnect" && action.sendTest {
						value["testMessage"] = map[string]any{"sent": false, "code": "test-target-unavailable"}
					}
					body, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
					if err != nil {
						return nil, err
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
				})
				store := &Store{
					client: &pluginrpc.Client{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{Transport: transport}},
					view:   View{Open: true, Available: true, Scope: 3, Channel: platform.ID},
					notify: func() {
						select {
						case notifications <- struct{}{}:
						default:
						}
					},
				}
				defer store.Hide()
				store.Review(3, action.name, "fixture-bot", "", "")
				if store.Snapshot().Review != action.name {
					t.Fatal("supported platform cannot review reconnect")
				}
				select {
				case req := <-calls:
					t.Fatal("review dispatched before confirmation", req)
				default:
				}
				store.Confirm()
				deadline := time.NewTimer(time.Second)
				defer deadline.Stop()
				for {
					view := store.Snapshot()
					if !view.Busy && len(calls) == 2 {
						if view.Error != "" {
							t.Fatal(view.Error)
						}
						if action.sendTest && view.TestNotice == "" {
							// finish(status) precedes publication of the reconnect receipt.
						} else {
							break
						}
					}
					select {
					case <-notifications:
					case <-deadline.C:
						t.Fatalf("reconnect did not settle: %+v", store.Snapshot())
					}
				}
				first, status := <-calls, <-calls
				if first.path != "/"+platform.ID+"/bot.reconnect" || first.request.Method != "bot.reconnect" || first.request.Payload["botId"] != "fixture-bot" || first.request.Payload["sendTest"] != action.sendTest || len(first.request.Payload) != 2 {
					t.Fatal("original channel/Bot/test payload changed", fmt.Sprint(first))
				}
				if status.path != "/"+platform.ID+"/connection.status" || status.request.Method != "connection.status" {
					t.Fatal("missing authoritative status refresh", status)
				}
				if !action.sendTest && store.Snapshot().TestNotice != "" {
					t.Fatal("plain reconnect fabricated a test receipt")
				}
			})
		}
	}
}
