package settings

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/dsh"
)

func TestLeavingServedFormDiscardsOnlyItsDraftsAndLiteral(t *testing.T) {
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:42123"}}, nil)
	s.writable = true
	s.namespaces = []Namespace{
		{NS: "agent-loop", Revision: 3, Value: json.RawMessage(`{"maxParallelToolCalls":4}`), Fields: []Field{{Key: "maxParallelToolCalls", Kind: FieldNumber}}},
		{NS: "llm-pi-ai", Revision: 3, Value: json.RawMessage(`{"providers":{}}`), Fields: []Field{{Key: "providers", Kind: FieldJSON}}},
	}
	loop := s.BindNamespace("agent-loop")
	loop.SetDraft("maxParallelToolCalls", "8")
	models := s.BindNamespace("llm-pi-ai")
	models.SetDraft("providers", `{"demo":{}}`)
	s.webSearchKey = &WebSearchCredential{state: WebSearchCredentialView{Draft: "synthetic-key"}}
	s.LeaveServedForms()
	state := s.BindNamespace("agent-loop").Snapshot()
	if state.Dirty || state.Drafts["maxParallelToolCalls"] != "4" || len(s.draftRevisions) != 1 || s.webSearchKey.state.Draft != "" || !models.Snapshot().Dirty {
		t.Fatalf("leave retained served edits or discarded another owner: %+v", state)
	}
}

func TestDocumentCapabilityCannotSurviveHostDetach(t *testing.T) {
	s := New()
	s.hasDocument = true
	if !s.Snapshot().HasDocument {
		t.Fatal("fixture capability missing")
	}
	s.Detach()
	if s.Snapshot().HasDocument {
		t.Fatal("stale Host document action remained available")
	}
}
