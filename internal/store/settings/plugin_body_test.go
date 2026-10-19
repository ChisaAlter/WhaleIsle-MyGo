package settings

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type namespaceTransport func(*http.Request) (*http.Response, error)

func (f namespaceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPluginNamespaceBodySavesBoundNamespaceWithoutSelectingRail(t *testing.T) {
	requests := make(chan map[string]any, 1)
	transport := namespaceTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		requests <- request
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"ns": "agent-loop", "revision": 8, "value": map[string]any{"parallelToolCalls": 8}, "base": map[string]any{"parallelToolCalls": 4}, "user": map[string]any{"parallelToolCalls": 8}, "secrets": []any{}}}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})
	s := New()
	saved := make(chan struct{}, 1)
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://host.fixture.invalid"}, HTTP: &http.Client{Transport: transport}}, func() {
		if s.Snapshot().LastAction == "settings saved" {
			select {
			case saved <- struct{}{}:
			default:
			}
		}
	})
	s.mu.Lock()
	s.writable = true
	s.selectedNS = "general"
	s.namespaces = []Namespace{{NS: "general", Value: json.RawMessage(`{}`)}, {NS: "agent-loop", Revision: 7, Value: json.RawMessage(`{"parallelToolCalls":4}`), Base: json.RawMessage(`{"parallelToolCalls":4}`), Fields: []Field{{Key: "parallelToolCalls", Kind: FieldNumber}}}}
	s.mu.Unlock()
	binding := s.BindNamespace("agent-loop")
	if binding == nil {
		t.Fatal("served namespace did not bind")
	}
	binding.SetForm("@provider-create", `{"route":"fixture"}`)
	if binding.Snapshot().Dirty || binding.Snapshot().Drafts["@provider-create"] == "" {
		t.Fatal("transient provider form became a settings write")
	}
	binding.SetDraft("parallelToolCalls", "8")
	if s.Snapshot().SelectedNS != "general" || binding.Snapshot().Drafts["parallelToolCalls"] != "8" {
		t.Fatal("bound draft changed the settings selection")
	}
	binding.Save(context.Background())
	select {
	case request := <-requests:
		args := request["payload"].(map[string]any)["args"].(map[string]any)
		if request["method"] != "settings/update" || args["ns"] != "agent-loop" || args["expectedRevision"] != float64(7) || args["patch"].(map[string]any)["parallelToolCalls"] != float64(8) {
			t.Fatalf("save borrowed another namespace/revision: %v", request)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("fixture save was not called")
	}
	select {
	case <-saved:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture save did not settle")
	}
	if s.Snapshot().SelectedNS != "general" || binding.Snapshot().Dirty {
		t.Fatal("save changed selection or failed to acknowledge its drafts")
	}
}

func TestPluginNamespaceOldCallbacksCannotWriteNewHost(t *testing.T) {
	var calls atomic.Int32
	transport := namespaceTransport(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		t.Error("stale binding crossed the RPC boundary")
		return nil, context.Canceled
	})
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://old.fixture.invalid"}, HTTP: &http.Client{Transport: transport}}, nil)
	s.mu.Lock()
	s.writable = true
	s.namespaces = []Namespace{{NS: "agent-loop", Fields: []Field{{Key: "parallelToolCalls", Kind: FieldNumber}}, Value: json.RawMessage(`{"parallelToolCalls":4}`)}}
	s.mu.Unlock()
	old := s.BindNamespace("agent-loop")
	old.SetDraft("parallelToolCalls", "8")
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://new.fixture.invalid"}, HTTP: &http.Client{Transport: transport}}, nil)
	old.SetDraft("parallelToolCalls", "99")
	old.SetForm("@provider-create", `{"route":"stale"}`)
	old.Reset("parallelToolCalls")
	old.Save(context.Background())
	if calls.Load() != 0 || old.Snapshot().ClientReady {
		t.Fatal("old namespace owner remained writable")
	}
	if s.Snapshot().Drafts["parallelToolCalls"] == "99" {
		t.Fatal("old field callback changed new Host drafts")
	}
}
