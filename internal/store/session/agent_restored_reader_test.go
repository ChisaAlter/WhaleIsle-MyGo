package session

import "testing"

type restoredAgentReader interface {
	RestoreRetainedAgentChat(string, string, AgentSessionAddress, string) bool
}

func TestRestoredAgentReaderRejectsUnboundTargets(t *testing.T) {
	root, _, address := retainedOwnerFixture()
	pane := root.NewRetainedAgentNavigationPane("restore")
	restore, ok := any(pane).(restoredAgentReader)
	if !ok {
		t.Fatal("missing verified restored-reader entrypoint")
	}
	for _, scope := range []string{"", "legacy", "wrong-host"} {
		if restore.RestoreRetainedAgentChat(scope, "root", address, "child") {
			t.Fatal("unbound restore admitted", scope)
		}
	}
	if pane.HasRetainedAgentRead() {
		t.Fatal("unverified restore acquired lease")
	}
}

func TestRestoredAgentReaderProjectionRequiresExactCatalog(t *testing.T) {
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
	for _, raw := range []string{`{}`, `{"subagentCatalog":[]}`, `{"subagentCatalog":[{"id":"child","mode":"one-shot"}]}`, `{"subagentCatalog":[{"id":"other","mode":"continuable"}]}`} {
		if restoredAgentProjectionContains("root", address, agentNavigationProjection{Values: []byte(raw)}) {
			t.Fatal("unproven relation", raw)
		}
	}
	if !restoredAgentProjectionContains("root", address, agentNavigationProjection{Values: []byte(`{"subagentCatalog":[{"id":"child","mode":"continuable"}]}`)}) {
		t.Fatal("exact live catalog rejected")
	}
}
func TestRestoredAgentReaderNeverGrantsActions(t *testing.T) {
	_, pane, address := retainedOwnerFixture()
	pane.agentRetainedRead.restoredReadOnly = true
	if !pane.RetainedAgentReadValid() {
		t.Fatal("read capability lost")
	}
	pane.mu.Lock()
	defer pane.mu.Unlock()
	if pane.agentPaneCurrentLocked() || pane.retainedAgentPromptAllowedLocked(7, address) {
		t.Fatal("automatic restore granted explicit action capability")
	}
}
