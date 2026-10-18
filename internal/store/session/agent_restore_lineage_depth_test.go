package session

import (
	"fmt"
	"testing"
)

func TestAgentRestoreLineageAcceptsFiniteDeepPath(t *testing.T) {
	path := make([]AgentSessionAddress, 80)
	parent := "root"
	for i := range path {
		child := fmt.Sprintf("child-%d", i)
		path[i] = AgentSessionAddress{Kind: "subagent", ParentSessionID: parent, ChildSessionID: child, Mode: "continuable"}
		parent = child
	}
	if !validAgentRestoreLineage("root", path[len(path)-1], path) {
		t.Fatal("finite acyclic saved lineage was truncated by arbitrary depth")
	}
	path[75].ChildSessionID = "root"
	if validAgentRestoreLineage("root", path[len(path)-1], path) {
		t.Fatal("cycle admitted")
	}
}
