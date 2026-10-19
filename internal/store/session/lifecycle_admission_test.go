package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

type lifecycleAdmissionTransport func(*http.Request) (*http.Response, error)

func (f lifecycleAdmissionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The real Client.Call and mutation worker run synchronously; no socket, sleeps,
// or inference from a transport's completion stands in for the result handler.
func TestLifecycleRepliesCannotSelectAcrossHostGeneration(t *testing.T) {
	for _, method := range []string{"session/create", "session/fork", "session/blankReuse", "scratch/create"} {
		for _, sameClient := range []bool{false, true} {
			for _, success := range []bool{false, true} {
				t.Run(method+map[bool]string{true: "/same-client", false: "/replacement"}[sameClient]+map[bool]string{true: "/success", false: "/failure"}[success], func(t *testing.T) {
					s := New()
					var client *dsh.Client
					client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://lifecycle-fixture.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
						rpcMethod := method
						if method == "scratch/create" {
							rpcMethod = "session/create"
						}
						if r.URL.Path != "/api/"+rpcMethod {
							t.Errorf("stale lifecycle started unexpected RPC %s", r.URL.Path)
						}
						var body struct {
							RPCID string `json:"rpcId"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							return nil, err
						}
						next := client
						if !sameClient {
							next = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}
						}
						s.Attach(next, nil)
						if sameClient {
							s.BeginEvents(context.Background(), client)
						}
						s.mu.Lock()
						s.selectedID = "current-host-session"
						s.lastAction = "current-host-status"
						s.mu.Unlock()
						if !success {
							return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("old Host refused"))}, nil
						}
						value := map[string]any{"sessionId": "old-host-result"}
						if method == "session/blankReuse" {
							value = map[string]any{"reusable": true}
						}
						raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
						return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
					})}}
					s.Attach(client, nil)
					generation := s.eventsGeneration
					switch method {
					case "session/create":
						s.runCreateSession(context.Background(), client, generation, "/owned", "")
					case "session/fork":
						s.runForkSession(context.Background(), client, generation, "owned-source")
					case "session/blankReuse":
						s.cwd = "/owned"
						s.items = []Item{{SessionID: "owned-blank", Cwd: "/owned", Blank: true}}
						s.CreateSession(context.Background())
					case "scratch/create":
						s.scratchCwd, s.workspaceReady = "/owned-scratch", true
						s.selectedID = "owned-scratch-source"
						s.runCreateNoDirectorySession(context.Background(), client, generation, s.selectedID, []string{"owned-scratch-source"})
					}
					got := s.Snapshot()
					if got.SelectedID != "current-host-session" || got.LastAction != "current-host-status" {
						t.Fatalf("old lifecycle polluted current Host: selected=%q action=%q", got.SelectedID, got.LastAction)
					}
				})
			}
		}
	}
}
