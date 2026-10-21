package frame

import (
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

// Clicking a tool card's inspect action must open the session log and queue a
// pick on the record that carries the call.
func TestInspectToolCallOpensSessionLogAtTheCallRecord(t *testing.T) {
	tab := "files"
	m := Model{
		RightPaneTab: &tab,
		Sizes:        &LayoutSizes{Workspace: &WorkspacePanes{Root: &WorkspacePane{ID: "p1", Kind: "files"}}},
	}
	m.inspectToolCall("msg-9", 33)
	if tab != "session-inspector-log" {
		t.Fatalf("right tab = %q, want the session log", tab)
	}
	panes := inspectorPaneStates(m.Sizes.Workspace)
	if len(panes) != 1 {
		t.Fatalf("inspector panes = %d, want the routed pane", len(panes))
	}
	pane := panes[0]
	if pane.Mode != "chat-node" {
		t.Fatalf("inspector mode = %q, want the chat projection", pane.Mode)
	}
	if pane.pendingPick == nil || pane.pendingPick.ID != "msg-9" || pane.pendingPick.Seq != 33 {
		t.Fatalf("pending pick = %+v", pane.pendingPick)
	}
}

// Developer tools off means no inspector seat: the card must hide the action
// rather than render a button that opens nothing.
func TestToolCallInspectorHiddenWithoutDeveloperTools(t *testing.T) {
	tab := "files"
	m := Model{RightPaneTab: &tab, DisableDeveloperTools: true,
		Sizes: &LayoutSizes{Workspace: &WorkspacePanes{}}}
	if m.toolCallInspector() != nil {
		t.Fatal("inspect action must stay nil when developer tools are off")
	}
}

// A queued pick waits for rows, then selects the real record and reports no miss.
func TestPendingPickWaitsForRowsThenSelectsTheRecord(t *testing.T) {
	p := &SessionInspectorPaneState{pendingPick: &chat.InspectorTarget{ID: "msg-9", Seq: 33}}
	p.applyPendingPick(nil)
	if p.pendingPick == nil {
		t.Fatal("pending pick dropped before any rows existed")
	}
	if p.navigation.Picked != nil {
		t.Fatal("pick resolved without a record")
	}
	rows := []session.InspectorRecord{
		{Key: "chat:msg-8", Seq: 30},
		{Key: "chat:msg-9", Seq: 33},
	}
	p.navigation.Nodes = []session.Message{{ID: "msg-9", Seq: 33}}
	p.applyPendingPick(rows)
	if p.navigation.Picked == nil || p.navigation.Picked.ID != "msg-9" {
		t.Fatalf("pick = %+v", p.navigation.Picked)
	}
	p.consumeInspectorPick(rows)
	if p.selected != "chat:msg-9" {
		t.Fatalf("selected row = %q", p.selected)
	}
	if p.navigation.Target == nil || p.navigation.Target.ID != "msg-9" {
		t.Fatalf("navigation target = %+v", p.navigation.Target)
	}
	if p.navigation.Notice != "" {
		t.Fatalf("unexpected miss notice: %q", p.navigation.Notice)
	}
}

// A record outside the loaded window reports the miss instead of selecting
// something else.
func TestPendingPickReportsMissingRecord(t *testing.T) {
	p := &SessionInspectorPaneState{
		pendingPick: &chat.InspectorTarget{ID: "msg-404", Seq: 99},
		navigation:  chat.InspectorNavigation{SessionID: "s1", Nodes: []session.Message{{ID: "msg-1"}}},
	}
	rows := []session.InspectorRecord{{Key: "chat:msg-1", Seq: 10}}
	p.applyPendingPick(rows)
	p.consumeInspectorPick(rows)
	if p.selected != "" {
		t.Fatalf("selected %q for a missing record", p.selected)
	}
	if p.navigation.Notice == "" {
		t.Fatal("missing record must be reported, not silently ignored")
	}
}

// The trajectory target wins when the chat tabs expose it; the session log is the
// fallback that keeps the action present in other layouts.
func TestInspectActionPrefersTrajectoryThenInspector(t *testing.T) {
	var got []string
	trajectory := func(callID string) { got = append(got, "traj:"+callID) }
	inspector := func(callID, messageID string, seq float64) { got = append(got, "insp:"+messageID) }
	inspectActionFor(trajectory, inspector)("call-1", "msg-1", 5)
	inspectActionFor(nil, inspector)("call-2", "msg-2", 6)
	if len(got) != 2 || got[0] != "traj:call-1" || got[1] != "insp:msg-2" {
		t.Fatalf("dispatch = %v", got)
	}
	if inspectActionFor(nil, nil) != nil {
		t.Fatal("no destination must stay nil so the card hides the action")
	}
}
