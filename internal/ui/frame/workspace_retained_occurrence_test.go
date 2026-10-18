package frame

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestSubagentChatOldOccurrenceCannotReleaseNewPaneID(t *testing.T) {
	owner := session.New()
	old := owner.NewRetainedAgentNavigationPane("same")
	fresh := owner.NewRetainedAgentNavigationPane("same")
	n := &WorkspacePane{ID: "same", Kind: "subagentchat", SubagentChat: &SubagentChatPaneState{Root: "old-root"}, Agents: &WorkspaceAgentState{Owner: owner, Navigation: old}}
	n.releaseAgentPane()
	if owner.AgentNavigationPaneByID(old.AgentNavigationOccurrenceID()) != nil {
		t.Fatal("old owner retained")
	}
	if owner.AgentNavigationPaneByID(fresh.AgentNavigationOccurrenceID()) != fresh {
		t.Fatal("same pane ID released replacement")
	}
}
