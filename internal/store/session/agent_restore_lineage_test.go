package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"whaleisle/internal/dsh"
)

type restoreLineageCapture interface {
	AgentChatRestoreLineage(uint64, AgentSessionAddress) []AgentSessionAddress
}

func TestAgentRestoreLineageCapturesOnlyVerifiedPath(t *testing.T) {
	s, a, leaf := nestedCatalogRoutingFixture()
	capture, ok := any(s).(restoreLineageCapture)
	if !ok {
		t.Fatal("missing verified lineage capture")
	}
	path := capture.AgentChatRestoreLineage(7, leaf)
	if len(path) != 2 || path[0] != a || path[1] != leaf {
		t.Fatalf("path %+v", path)
	}
	if len(capture.AgentChatRestoreLineage(6, leaf)) != 0 {
		t.Fatal("stale token captured path")
	}
	s.agentNavigation.view.Rows = nil
	if len(capture.AgentChatRestoreLineage(7, leaf)) != 0 {
		t.Fatal("removed ancestor captured")
	}
}

func TestAgentRestoreLineageRejectsCyclesAndMissingPath(t *testing.T) {
	_, a, leaf := nestedCatalogRoutingFixture()
	if validAgentRestoreLineage("root", leaf, nil) || validAgentRestoreLineage("other", leaf, []AgentSessionAddress{a, leaf}) {
		t.Fatal("unproven root accepted")
	}
	cycle := leaf
	cycle.ChildSessionID = "root"
	if validAgentRestoreLineage("root", cycle, []AgentSessionAddress{a, cycle}) {
		t.Fatal("cycle accepted")
	}
	if !validAgentRestoreLineage("root", leaf, []AgentSessionAddress{a, leaf}) {
		t.Fatal("valid ID chain rejected")
	}
}
func TestAgentRestoreLineageStopsAtFirstUnverifiedEdge(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	a := address
	a.ChildSessionID = "a"
	b := address
	b.ParentSessionID = "a"
	b.ChildSessionID = "b"
	last := address
	last.ParentSessionID = "b"
	var visited []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					Request struct {
						SessionID string `json:"sessionId"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		id := body.Payload.Args.Request.SessionID
		visited = append(visited, id)
		entries := []any{}
		if id == "root" {
			entries = []any{map[string]any{"id": "a", "mode": "continuable"}}
		}
		rpcOK(w, body.RPCID, map[string]any{"values": map[string]any{"subagentCatalog": entries}})
	}))
	defer server.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	root.client, pane.client = client, client
	lease := pane.agentRetainedRead
	lease.client = client
	lease.address = last
	lease.lineage = []AgentSessionAddress{a, b, last}
	lease.restoredReadOnly = true
	pane.agentNavigation.view.Address = last
	pane.verifyRestoredAgentReader(lease)
	if !reflect.DeepEqual(visited, []string{"root", "a"}) {
		t.Fatalf("not bounded edge validation: %v", visited)
	}
	if lease.ctx.Err() == nil {
		t.Fatal("failed edge retained capability")
	}
}
func TestAgentRestoreLineageIntermediateDeletionCancels(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	a := address
	a.ChildSessionID = "ancestor"
	pane.agentRetainedRead.lineage = []AgentSessionAddress{a}
	root.mu.Lock()
	root.cancelRetainedAgentReadsLocked("ancestor")
	root.mu.Unlock()
	if pane.agentRetainedRead.ctx.Err() == nil {
		t.Fatal("intermediate ancestor removal did not revoke")
	}
}
