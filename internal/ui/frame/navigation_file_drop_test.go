package frame

import (
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func TestNavigationFileDropOwner(t *testing.T) {
	owner := session.New()
	d := composer.NewDraftState("")
	w := &WorkspacePanes{}
	m := Model{Store: owner, Draft: &d.Text, ComposerState: d, OnAttachComposerFiles: func([]string) {}, Sizes: &LayoutSizes{Workspace: w}}
	if !navigationFileDropAllowed(m) {
		t.Fatal("main draft rejected")
	}
	w.conversation = &mainChildConversation{Owner: owner, Binding: owner.Snapshot().PlanHistoryBinding, Pane: &WorkspacePane{SubagentChat: &SubagentChatPaneState{Root: owner.Snapshot().SelectedID}}}
	if navigationFileDropAllowed(m) {
		t.Fatal("child main view accepted parent attachment")
	}
	w.conversation = nil
	m.OnAttachComposerFiles = nil
	if navigationFileDropAllowed(m) {
		t.Fatal("missing owner intake registered target")
	}
}
