package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestInspectToolCallOpensAndSelectsRawTrajectory(t *testing.T) {
	tab, selected, query, detail := "chat", "old", "hidden", "result"
	turns, calls := true, true
	list := &ui.ListState{FollowEnd: true}
	m := Model{ChatPaneTab: &tab, TrajSelectedID: &selected, TrajSearch: &query, TrajDetailTab: &detail, TrajTurnsCollapsed: &turns, TrajCallsCollapsed: &calls, ChatList: list}
	snap := session.ViewState{Messages: []session.Message{{ID: "raw-call", Kind: session.MessageKindTool, Raw: []byte(`{"type":"tool/call","data":{"callId":"c"}}`)}}}
	if !focusToolCallTrajectory(m, snap, "c") || tab != "trajectory" || selected != "raw-call" || query != "" || turns || calls || detail != "args" || list.FollowEnd {
		t.Fatal("inspect did not unfold/focus real row")
	}
	if focusToolCallTrajectory(m, snap, "missing") || selected != "raw-call" {
		t.Fatal("missing call changed selection")
	}
}

func TestInspectToolCallRejectsStaleSessionAndUnavailableTabs(t *testing.T) {
	store := session.New()
	tab, selected := "chat", "old"
	m := Model{Store: store, ChatPaneTab: &tab, TrajSelectedID: &selected}
	callback := inspectToolCallNavigation(m, session.ViewState{SelectedID: "stale"}, &pendingToolCall{})
	callback("c")
	if tab != "chat" || selected != "old" {
		t.Fatal("stale session navigated")
	}
	m.HideChatViewTabs = true
	if inspectToolCallNavigation(m, session.ViewState{}, &pendingToolCall{}) != nil {
		t.Fatal("hidden trajectory exposed")
	}
}
