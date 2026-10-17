package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestAgentPresetDraftScope(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.cwd = "/one"
	s.agentPresetOptions = []AgentPresetOption{{ID: "a", Label: "Same"}, {ID: "b", Label: "Same"}}
	s.SelectAgentPresetIDForSession(context.Background(), "", "b")
	if got := s.AgentPresetSelection(); got.ID != "b" || got.Label != "Same" {
		t.Fatalf("not staged by ID: %+v", got)
	}
	owner, err := s.CaptureComposerTarget("")
	if err != nil {
		t.Fatal(err)
	}
	if owner.draftPresetID != "b" {
		t.Fatal("gesture lost preset")
	}
	s.cwd = "/two"
	if got := s.AgentPresetSelection(); got.ID != "" {
		t.Fatal("leaked to workspace")
	}
	s.cwd = "/one"
	s.composerHostGeneration++
	if got := s.AgentPresetSelection(); got.ID != "" {
		t.Fatal("leaked to Host")
	}
}
func TestAgentPresetStartedSessionRefuses(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "used"
	s.items = []Item{{SessionID: "used", Blank: false}}
	s.agentPresetOptions = []AgentPresetOption{{ID: "a", Label: "Coder"}}
	s.SelectAgentPresetIDForSession(context.Background(), "used", "a")
	if got := s.AgentPresetSelection(); got.CanSelect || got.Error == "" || got.Busy {
		t.Fatalf("started-session mutation allowed: %+v", got)
	}
}

func TestAgentPresetDraftCreationWire(t *testing.T) {
	var preset string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					Request struct {
						AgentPreset string `json:"agentPreset"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		preset = b.Payload.Args.Request.AgentPreset
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": b.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"sessionId": "new"}}})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.cwd = "/one"
	s.agentPresetOptions = []AgentPresetOption{{ID: "opaque-b", Label: "Same"}}
	s.SelectAgentPresetIDForSession(context.Background(), "", "opaque-b")
	owner, _ := s.CaptureComposerTarget("")
	s.cwd = "/two"
	id, err := owner.CreateSession(context.Background(), "/one")
	if err != nil || id != "new" || preset != "opaque-b" {
		t.Fatalf("creation %q %q %v", id, preset, err)
	}
	s.cwd = "/one"
	if s.AgentPresetSelection().ID != "" {
		t.Fatal("staged choice not consumed")
	}
}

func TestAgentPresetSelectionSingleFlightAndRefusal(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		calls.Add(1)
		close(entered)
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": b["rpcId"], "result": map[string]any{"ok": false, "error": map[string]any{"message": "fixture refusal"}}})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "blank"
	s.items = []Item{{SessionID: "blank", Blank: true}}
	s.agentPresetOptions = []AgentPresetOption{{ID: "a", Label: "Coder"}}
	s.SelectAgentPresetIDForSession(context.Background(), "blank", "a")
	<-entered
	if !s.AgentPresetSelection().Busy {
		t.Fatal("not pending")
	}
	s.SelectAgentPresetIDForSession(context.Background(), "blank", "a")
	close(release)
	deadline := time.Now().Add(time.Second)
	for s.AgentPresetSelection().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := s.AgentPresetSelection(); got.Busy || !strings.Contains(got.Error, "Coder") || calls.Load() != 1 {
		t.Fatalf("feedback %+v calls %d", got, calls.Load())
	}
}
