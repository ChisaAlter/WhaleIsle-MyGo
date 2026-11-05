package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestDefaultPresetTargetRejectsReplacementAndUsedSessions(t *testing.T) {
	for _, transition := range []string{"Host", "directory", "away-and-back", "used", "running", "empty"} {
		t.Run(transition, func(t *testing.T) {
			s := New()
			s.client = &dsh.Client{}
			s.selectedID, s.cwd, s.followGeneration = "blank", "/one", 3
			s.items = []Item{{SessionID: "blank", Blank: true}}
			s.agentPresetOptions = []AgentPresetOption{{ID: "a", Label: "A"}}
			apply := s.CaptureBlankAgentPresetDefaultTarget()
			if apply == nil {
				t.Fatal("real blank session was not captured")
			}
			switch transition {
			case "Host":
				s.client = &dsh.Client{}
			case "directory":
				s.cwd = "/two"
			case "away-and-back":
				s.followGeneration += 2
			case "used":
				s.items[0].Blank = false
			case "running":
				s.items[0].Running = true
			case "empty":
				s.selectedID = ""
			}
			apply(context.Background(), "a")
			if s.presetDraft.busy || len(s.presetDraft.staged) > 0 {
				t.Fatal("captured default targeted a replacement session or future draft")
			}
		})
	}
	s := New()
	s.client = &dsh.Client{}
	if s.CaptureBlankAgentPresetDefaultTarget() != nil {
		t.Fatal("absence of a session became a mutable target")
	}
}

func TestDefaultPresetTargetSelectsOriginalBlankSession(t *testing.T) {
	request := make(chan map[string]any, 1)
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: presetRoundTrip(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		value := any(map[string]any{})
		if body["method"] == "agentPresets/select" {
			request <- body["payload"].(map[string]any)["args"].(map[string]any)
			value = "a"
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	s.selectedID, s.cwd, s.followGeneration = "blank", "/one", 3
	s.items = []Item{{SessionID: "blank", Blank: true}}
	s.agentPresetOptions = []AgentPresetOption{{ID: "a", Label: "A"}}
	s.CaptureBlankAgentPresetDefaultTarget()(context.Background(), "a")
	select {
	case args := <-request:
		if args["agentId"] != "blank" || args["agentPreset"] != "a" {
			t.Fatalf("wrong selected target: %+v", args)
		}
	case <-time.After(time.Second):
		t.Fatal("captured blank session was not selected")
	}
}
