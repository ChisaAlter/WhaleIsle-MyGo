package frame

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestToolCallPendingWaitsForLoadedSnapshot(t *testing.T) {
	tab, selected := "chat", "old"
	m := Model{ChatPaneTab: &tab, TrajSelectedID: &selected}
	snapshot := session.ViewState{SelectedID: "a", ComposerHostGeneration: 3}
	pending := pendingToolCall{}
	pending.request(m, snapshot, "later")
	if tab != "trajectory" || pending.callID != "later" || selected != "" {
		t.Fatal("missing call not retained")
	}
	pending.apply(m, snapshot)
	if pending.callID != "later" {
		t.Fatal("empty snapshot consumed request")
	}
	snapshot.Messages = []session.Message{{ID: "arrived", Kind: session.MessageKindTool, Tool: &session.ToolCallView{CallID: "later"}}}
	pending.apply(m, snapshot)
	if selected != "arrived" || pending.callID != "" {
		t.Fatal("loaded data did not fulfill request")
	}
	selected = "manual-selection"
	pending.apply(m, snapshot)
	if selected != "manual-selection" {
		t.Fatal("fulfilled request stole selection")
	}
}

func TestToolCallPendingReleasesOnSessionHostOrTabChange(t *testing.T) {
	for _, change := range []string{"session", "host", "tab"} {
		tab, selected := "chat", ""
		m := Model{ChatPaneTab: &tab, TrajSelectedID: &selected}
		snapshot := session.ViewState{SelectedID: "a", ComposerHostGeneration: 3}
		pending := pendingToolCall{}
		pending.request(m, snapshot, "later")
		switch change {
		case "session":
			snapshot.SelectedID = "b"
		case "host":
			snapshot.ComposerHostGeneration++
		case "tab":
			tab = "chat"
		}
		pending.apply(m, snapshot)
		if pending.callID != "" {
			t.Fatalf("%s left pending request", change)
		}
	}
}
