package main

import (
	"testing"
	"whaleisle/internal/store/session"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
)

func TestAgentDocumentPaneOwner(t *testing.T) {
	root := session.New()
	state := &mainState{sessions: root}
	a, b := root.AgentNavigationPane("a"), root.AgentNavigationPane("b")
	source := browserpreviewui.DocumentSource{AddressKind: "subagent", NavigationPaneID: "a"}
	if state.workspaceDocumentStore(source) != a {
		t.Fatal("wrong addressed owner")
	}
	root.ReleaseAgentNavigationPane("a")
	if state.workspaceDocumentStore(source) != nil {
		t.Fatal("borrowed another pane after source closed")
	}
	source.NavigationPaneID = "b"
	if state.workspaceDocumentStore(source) != b {
		t.Fatal("other source invalidated")
	}
}
