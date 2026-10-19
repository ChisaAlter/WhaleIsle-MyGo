package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/workflowrun"
)

// The real session/list envelope runs entirely through an in-memory transport.
// No listener, Host, child follow, model, or user workspace enters this fixture.
func TestSubagentIdentityListControlSparseAndHostReset(t *testing.T) {
	s := New()
	s.applySessionSubagentIdentityControlLocked(json.RawMessage(`{"type":"baseline","value":{"projections":{"child":{"asOfSeq":10,"values":{"subagent":{"mode":"continuable","label":"Owned child","seq":3}}},"later":{"asOfSeq":4,"values":{"subagent":{"mode":"one-shot","seq":2}}}}}}`))
	rows := []api.SessionListValueItemsItem{{SessionId: "child", Running: true, Projections: &api.SessionListValueItemsItemProjections{Kind: "cached", AsOfSeq: 9999, Values: json.RawMessage(`{"subagent":{"mode":"one-shot","seq":1}}`)}}}
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://identity-fixture.invalid"}, HTTP: &http.Client{Transport: composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/session/list" {
			return nil, fmt.Errorf("unexpected fixture method %s", r.URL.Path)
		}
		return composerRegressionReply(r, api.SessionListValue{Items: rows})
	})}}
	refresh := func() {
		s.mu.Lock()
		client, generation := s.beginListRefreshLocked()
		s.mu.Unlock()
		s.runRefreshList(context.Background(), client, generation)
	}
	assertMode := func(id, want string) {
		t.Helper()
		for _, row := range s.Snapshot().Items {
			if row.SessionID == id {
				mode := ""
				if row.SubagentIdentity != nil {
					mode = row.SubagentIdentity.Mode
				}
				if mode != want {
					t.Fatalf("%s mode=%q want=%q: %+v", id, mode, want, row.SubagentIdentity)
				}
				return
			}
		}
		t.Fatalf("missing roster row %s", id)
	}
	refresh()
	assertMode("child", "continuable") // Control preceded its first roster row.
	s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/added", sessionID: "later", item: Item{SessionID: "later", Running: true}}, true)
	assertMode("later", "one-shot")
	s.applySessionSubagentIdentityControlLocked(json.RawMessage(`{"type":"projection","sessionId":"child","key":"subagent","seq":11,"value":null}`))
	assertMode("child", "")
	refresh() // Numerically higher cold cached data cannot resurrect identity.
	assertMode("child", "")
	s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/added", sessionID: "child", item: Item{SessionID: "child", Running: true}}, true)
	assertMode("child", "")
	s.applySessionSubagentIdentityControlLocked(json.RawMessage(`{"type":"projection","sessionId":"child","key":"subagent","seq":12,"value":{"mode":"one-shot","seq":8}}`))
	s.applySessionSubagentIdentityControlLocked(json.RawMessage(`{"type":"projection","sessionId":"child","key":"subagent","seq":10,"value":{"mode":"continuable","seq":3}}`))
	s.applySessionSubagentIdentityControlLocked(json.RawMessage(`{"type":"projection","sessionId":"child","key":"subagent","value":null}`))
	assertMode("child", "one-shot")
	s.applySessionSubagentIdentityValuesLocked("child", "sequenced", 13, json.RawMessage(`{"unrelatedProjection":{}}`))
	assertMode("child", "one-shot")
	s.selectedID = "child"
	s.applyFollowFrameLocked(json.RawMessage(`{"type":"snapshot","cursor":14,"records":[],"projections":{"asOfSeq":14,"values":{"subagent":{"mode":"continuable","label":"Own descriptor","seq":13}}}}`))
	assertMode("child", "continuable")
	// A replayed descriptor is not proof of an own-log suffix. Only the Host's
	// folded projection may change the identity, including its explicit null.
	s.applyPersistenceEventLocked(json.RawMessage(`{"type":"subagent/descriptor","seq":15,"data":{"mode":"one-shot"}}`))
	assertMode("child", "continuable")
	s.applySessionSubagentIdentityControlLocked(json.RawMessage(`{"type":"projection","sessionId":"child","key":"subagent","seq":16,"value":{"mode":"unknown","seq":15}}`))
	assertMode("child", "")
	s.Attach(&dsh.Client{}, nil)
	assertMode("child", "")
	if len(s.subagentIdentities) != 0 {
		t.Fatal("Host replacement retained a detached identity cache")
	}
}

func TestSubagentIdentityRemovalRetainsResidentButDeletionForgetsOwner(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "child", ParentID: "parent", Origin: "subagent", Running: true}, {SessionID: "other"}}
	s.applySessionSubagentIdentityValuesLocked("child", "sequenced", 5, json.RawMessage(`{"subagent":{"mode":"continuable","label":"Child","seq":3}}`))
	s.applySessionSubagentIdentityValuesLocked("other", "sequenced", 8, json.RawMessage(`{"subagent":{"mode":"one-shot","seq":4}}`))
	s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/removed", sessionID: "child"}, true)
	if s.items[0].SubagentIdentity == nil || s.items[0].SubagentIdentity.Mode != "continuable" || s.items[0].Running {
		t.Fatal("ordinary registry removal discarded durable child identity or kept it live")
	}
	s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/deleted", sessionID: "child"}, true)
	s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/added", sessionID: "child", item: Item{SessionID: "child", ParentID: "parent", Origin: "subagent"}}, true)
	for _, item := range s.items {
		if item.SessionID == "child" && item.SubagentIdentity != nil {
			t.Fatal("a durable-deleted owner leaked its old identity into a sparse new row")
		}
		if item.SessionID == "other" && item.SubagentIdentity == nil {
			t.Fatal("deleting one owner cleared another child's identity")
		}
	}
	s.Detach()
	if len(s.subagentIdentities) != 0 || s.items[0].SubagentIdentity != nil {
		t.Fatal("detach retained Host-scoped identity")
	}
}

func TestWorkflowNavigationUsesAuthoritativeModeAndCurrentMemberOwner(t *testing.T) {
	for _, test := range []struct {
		name, mode string
		mutate     func(*Store)
		want       bool
	}{
		{"one shot", "one-shot", nil, true},
		{"continuable", "continuable", nil, true},
		{"unknown", "unknown", nil, false},
		{"external", "external", nil, false},
		{"missing identity", "", func(s *Store) { s.items[0].SubagentIdentity = nil }, false},
		{"bare origin is insufficient", "", func(s *Store) { s.items[0].SubagentIdentity = nil; s.items[0].Origin = "subagent" }, false},
		{"foreign parent", "one-shot", func(s *Store) { s.items[0].ParentID = "foreign" }, false},
		{"idle child", "one-shot", func(s *Store) { s.items[0].Running = false }, false},
		{"changed selection", "one-shot", func(s *Store) { s.selectedID = "foreign" }, false},
		{"settled member", "one-shot", func(s *Store) {
			s.messages = append(s.messages, Message{Raw: json.RawMessage(`{"type":"tool-workflow/agent-end","seq":6,"data":{"runId":"r","seq":0,"outcome":"completed"}}`)})
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := New()
			s.selectedID, s.planHistoryBinding = "parent", "exact-binding"
			s.items = []Item{{SessionID: "child", ParentID: "parent", Running: true, SubagentIdentity: &SessionSubagentIdentity{Mode: test.mode, Seq: 3}}}
			s.messages = []Message{
				{Raw: json.RawMessage(`{"type":"tool-workflow/run-start","seq":3,"data":{"runId":"r","name":"Build"}}`)},
				{Raw: json.RawMessage(`{"type":"tool-workflow/agent-start","seq":4,"data":{"runId":"r","seq":0,"label":"Member","childId":"child"}}`)},
			}
			if test.mutate != nil {
				test.mutate(s)
			}
			address, allowed := s.workflowMemberAddressLocked("parent", "r", 0, "child")
			if allowed != test.want {
				t.Fatalf("member mode/owner gate=%v want=%v address=%v", allowed, test.want, address)
			}
			if allowed {
				want := map[string]any{"kind": "subagent", "parentSessionId": "parent", "childSessionId": "child", "mode": test.mode}
				if !reflect.DeepEqual(address, want) {
					t.Fatalf("mode or owner was guessed: %v", address)
				}
				wire, _ := json.Marshal(workflowMemberFollowPayload(address))
				var envelope struct {
					Args struct {
						Request struct {
							Address         map[string]any `json:"address"`
							AssistantStream bool           `json:"assistantStream"`
						} `json:"request"`
					} `json:"args"`
				}
				if json.Unmarshal(wire, &envelope) != nil || !envelope.Args.Request.AssistantStream || !reflect.DeepEqual(envelope.Args.Request.Address, want) {
					t.Fatalf("real follow payload lost identity mode: %s", wire)
				}
			}
			for _, stale := range []struct {
				run, child string
				seq        float64
			}{{"other-run", "child", 0}, {"r", "other-child", 0}, {"r", "child", 1}} {
				if _, allowed := s.workflowMemberAddressLocked("parent", stale.run, stale.seq, stale.child); allowed {
					t.Fatal("stale workflow member acquired an address")
				}
			}
			// Exercise the public entrypoint without a socket: missing client or
			// mux is still a closed gate, even with a valid authoritative mode.
			s.OpenWorkflowMember("exact-binding", "parent", "r", 0, "child", "Member")
			if s.WorkflowMemberSnapshot().Open {
				t.Fatal("workflow opened without the current Host transport owner")
			}
		})
	}
	if WorkflowMemberAvailable([]Item{{SessionID: "child", ParentID: "parent", Running: true}}, "parent", workflowrun.Member{Status: "running", ChildID: "child"}) {
		t.Fatal("ordinary running child without descriptor identity was navigable")
	}
}

func TestOpenWorkflowMemberCapturesActualModeWithoutRootSelection(t *testing.T) {
	for _, mode := range []string{"one-shot", "continuable"} {
		t.Run(mode, func(t *testing.T) {
			s := New()
			s.selectedID, s.planHistoryBinding = "parent", "exact-binding"
			s.client, s.mux = &dsh.Client{}, &dsh.Mux{}
			s.items = []Item{{SessionID: "child", ParentID: "parent", Running: true, SubagentIdentity: &SessionSubagentIdentity{Mode: mode, Seq: 3}}}
			s.messages = []Message{
				{Raw: json.RawMessage(`{"type":"tool-workflow/run-start","seq":3,"data":{"runId":"r","name":"Build"}}`)},
				{Raw: json.RawMessage(`{"type":"tool-workflow/agent-start","seq":4,"data":{"runId":"r","seq":0,"label":"Member","childId":"child"}}`)},
			}
			var captured map[string]any
			s.notify = func() {
				s.mu.Lock()
				captured = s.workflowMember.address
				cancel := s.workflowMember.cancel
				s.mu.Unlock()
				// Open notifies synchronously before starting its observer. Cancel
				// at that boundary: the production startup guard exits before Dial,
				// with no fake listener or real network involved in this test.
				if cancel != nil {
					cancel()
				}
			}
			s.OpenWorkflowMember("stale-binding", "parent", "r", 0, "child", "Member")
			if captured != nil || s.WorkflowMemberSnapshot().Open {
				t.Fatal("old history binding opened a workflow observer")
			}
			s.OpenWorkflowMember("exact-binding", "parent", "r", 0, "child", "Member")
			want := map[string]any{"kind": "subagent", "parentSessionId": "parent", "childSessionId": "child", "mode": mode}
			if !reflect.DeepEqual(captured, want) || s.Snapshot().SelectedID != "parent" {
				t.Fatalf("public workflow open lost captured child mode/parent: %v", captured)
			}
			s.mu.Lock()
			s.notify = nil
			s.mu.Unlock()
			s.CloseWorkflowMember(s.WorkflowMemberSnapshot().Token)
		})
	}
}
