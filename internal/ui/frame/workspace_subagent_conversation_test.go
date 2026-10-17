package frame

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestMainChildConversationLifetime(t *testing.T) {
	owner := session.New()
	w := &WorkspacePanes{}
	n := &WorkspacePane{ID: "main-child", Kind: "subagentchat", SubagentChat: &SubagentChatPaneState{Root: "root"}}
	n.Agents = &WorkspaceAgentState{Owner: owner, Navigation: owner.AgentNavigationPane(n.ID)}
	w.conversation = &mainChildConversation{Pane: n, Owner: owner, Binding: "old-host"}
	m := Model{Store: owner, Sizes: &LayoutSizes{Workspace: w}}
	if w.currentMainChild(m) != nil {
		t.Fatal("root/Host mismatch retained main child")
	}
	if w.conversation != nil || owner.AgentNavigationPaneByID(n.ID) != nil {
		t.Fatal("stale child capability retained")
	}
	subagentConversationOpener(m)(0, session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "forged", Mode: "continuable"})
	if w.conversation != nil {
		t.Fatal("uncatalogued address opened main conversation")
	}
}

func TestMainChildConversationReturnAndPause(t *testing.T) {
	owner := session.New()
	w := &WorkspacePanes{}
	owner.Select("root")
	n := &WorkspacePane{ID: "main-child", Kind: "subagentchat", SubagentChat: &SubagentChatPaneState{Root: "root", Address: session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}}}
	w.conversation = &mainChildConversation{Pane: n, Owner: owner}
	m := Model{Store: owner, Sizes: &LayoutSizes{Workspace: w}}
	actions := map[string]func(){"session.archive": func() {}, "session.rename": func() {}, "session.fork": func() {}, "session.search": func() {}}
	guardMainChildCommands(m, actions)
	if len(actions) != 1 || actions["session.search"] == nil {
		t.Fatal("hidden parent commands were not scoped")
	}
	if strings.Contains(string(w.Export()), "main-child") {
		t.Fatal("live main child route was persisted")
	}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { mainSubagentConversation(c, m) }) }, 600, 400)
	if n.Agents == nil || n.Agents.Navigation == nil {
		t.Fatal("route did not acquire independent navigation")
	}
	w.BeginContentFrame("file-binding", "/workspace")
	if w.conversation == nil {
		t.Fatal("filesystem binding incorrectly revoked session route")
	}
	w.EndContentFrame()
	if n.Agents != nil && n.Agents.Navigation != nil {
		t.Fatal("hidden route retained reader")
	}
	// Render again: it remains paused until an explicit reopen, never auto-follow.
	tt.Move(590, 390)
	r, ok := tt.Find("返回父会话")
	if !ok {
		r, ok = tt.Find("Back to parent conversation")
	}
	if !ok {
		t.Fatal("missing parent return")
	}
	tt.Press(r.X+5, r.Y+5)
	tt.Release(r.X+5, r.Y+5)
	if w.conversation != nil || n.Agents != nil && n.Agents.Navigation != nil {
		t.Fatal("return did not release child")
	}
	if owner.Snapshot().SelectedID != "root" {
		t.Fatal("child changed the parent top-level session")
	}
}
