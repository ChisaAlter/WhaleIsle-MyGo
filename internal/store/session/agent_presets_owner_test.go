package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type presetRoundTrip func(*http.Request) (*http.Response, error)

func (f presetRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type presetReplyBody struct {
	io.Reader
	closed chan struct{}
}

func (b *presetReplyBody) Close() error { close(b.closed); return nil }

func TestAgentPresetCatalogRejectsRetiredReplies(t *testing.T) {
	for _, transition := range []string{"attach-other", "detach", "same-client-generation", "newer-request"} {
		for _, failed := range []bool{false, true} {
			label := transition + "/success"
			if failed {
				label = transition + "/error"
			}
			t.Run(label, func(t *testing.T) {
				entered, release, bodyClosed := make(chan struct{}), make(chan struct{}), make(chan struct{})
				var calls atomic.Int32
				rt := presetRoundTrip(func(r *http.Request) (*http.Response, error) {
					var req struct {
						RPCID string `json:"rpcId"`
					}
					if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
						return nil, e
					}
					call := calls.Add(1)
					if call == 1 {
						close(entered)
						<-release
					}
					result := map[string]any{"ok": true, "value": map[string]any{"presets": []map[string]any{{"id": "current", "isDefault": true}}}}
					if call == 1 {
						result = map[string]any{"ok": true, "value": map[string]any{"presets": []map[string]any{{"id": "retired", "isDefault": true}}}}
						if failed {
							result = map[string]any{"ok": false, "error": map[string]any{"code": "fixture/retired", "message": "retired owner"}}
						}
					}
					raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": result})
					closed := make(chan struct{})
					if call == 1 {
						closed = bodyClosed
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: &presetReplyBody{Reader: strings.NewReader(string(raw)), closed: closed}}, nil
				})
				client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://owned-fixture.invalid"}, HTTP: &http.Client{Transport: rt}}
				s := New()
				s.Attach(client, nil)
				s.RefreshAgentPresets(context.Background())
				select {
				case <-entered:
				case <-time.After(time.Second):
					t.Fatal("read not started")
				}
				switch transition {
				case "attach-other":
					s.Attach(&dsh.Client{Endpoint: client.Endpoint, HTTP: client.HTTP}, nil)
				case "detach":
					s.Attach(nil, nil)
				case "same-client-generation":
					s.mu.Lock()
					s.eventsGeneration++
					s.mu.Unlock()
				case "newer-request":
					s.RefreshAgentPresets(context.Background())
				}
				if transition == "newer-request" {
					until := time.Now().Add(time.Second)
					for time.Now().Before(until) {
						s.mu.Lock()
						ready := len(s.agentPresetOptions) == 1 && s.agentPresetOptions[0].ID == "current"
						s.mu.Unlock()
						if ready {
							break
						}
						time.Sleep(time.Millisecond)
					}
				} else {
					s.mu.Lock()
					s.agentPresetOptions = []AgentPresetOption{{ID: "current", Label: "current"}}
					s.selectedAgentPresetLabel = "current"
					s.agentPresetsErr = nil
					s.mu.Unlock()
				}
				close(release)
				select {
				case <-bodyClosed:
				case <-time.After(time.Second):
					t.Fatal("old response not consumed")
				}
				until := time.NewTimer(75 * time.Millisecond)
				defer until.Stop()
				tick := time.NewTicker(time.Millisecond)
				defer tick.Stop()
				for {
					s.mu.Lock()
					valid := len(s.agentPresetOptions) == 1 && s.agentPresetOptions[0].ID == "current" && s.selectedAgentPresetLabel == "current" && s.agentPresetsErr == nil
					s.mu.Unlock()
					if !valid {
						t.Fatal("retired success/error changed current choices/default/error")
					}
					select {
					case <-until.C:
						return
					case <-tick.C:
					}
				}
			})
		}
	}
}

func TestAgentPresetCatalogCanceledBeforeStartDoesNotRead(t *testing.T) {
	var calls atomic.Int32
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://owned-fixture.invalid"}, HTTP: &http.Client{Transport: presetRoundTrip(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, context.Canceled })}}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.RefreshAgentPresets(ctx)
	if calls.Load() != 0 {
		t.Fatal("canceled refresh started a request")
	}
}

func TestAgentPresetCatalogCacheBelongsToAttachedHost(t *testing.T) {
	for _, transition := range []string{"other-client", "attach-nil", "detach"} {
		t.Run(transition, func(t *testing.T) {
			s := New()
			s.Attach(&dsh.Client{}, nil)
			s.agentPresetOptions = []AgentPresetOption{{ID: "old-preset", Label: "Old preset"}}
			s.selectedAgentPresetLabel = "Old preset"
			s.agentPresetsErr = errors.New("old Host refusal")
			switch transition {
			case "other-client":
				s.Attach(&dsh.Client{}, nil)
			case "attach-nil":
				s.Attach(nil, nil)
			case "detach":
				s.Detach()
			}
			got := s.Snapshot()
			if len(got.AgentPresetLabels) != 0 || got.SelectedAgentPreset != "" || got.AgentPresetsErr != "" {
				t.Fatalf("old Host catalog remains selectable: %+v %q %q", got.AgentPresetLabels, got.SelectedAgentPreset, got.AgentPresetsErr)
			}
			// Empty-session selection is a local future-session draft, so this
			// exposes stale menu admission without any Host or model request.
			s.SelectAgentPresetForSession(context.Background(), "Old preset")
			if v := s.AgentPresetSelection(); v.ID != "" {
				t.Fatal("old Host option staged on current Host", v)
			}
		})
	}
	client := &dsh.Client{}
	s := New()
	s.Attach(client, nil)
	s.agentPresetOptions = []AgentPresetOption{{ID: "current", Label: "Current"}}
	s.selectedAgentPresetLabel = "Current"
	s.selectedID = "retained-session"
	s.Attach(client, nil)
	if got := s.Snapshot(); len(got.AgentPresetLabels) != 1 || got.SelectedAgentPreset != "Current" || got.SelectedID != "retained-session" {
		t.Fatal("same-owner Attach discarded current choices or selected identity")
	}
	s.Attach(&dsh.Client{}, nil)
	if s.Snapshot().SelectedID != "retained-session" {
		t.Fatal("catalog reset changed selected session identity")
	}
}
