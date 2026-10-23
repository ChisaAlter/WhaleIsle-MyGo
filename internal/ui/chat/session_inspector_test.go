package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestInspectorChatIdentityAndFoldReveal(t *testing.T) {
	msgs := []session.Message{{ID: "first", Seq: 10}, {ID: "exact", Seq: 20}, {ID: "last", Seq: 30}}
	if inspectorMessageIndex(msgs, InspectorTarget{ID: "exact", Seq: 30}) != 1 {
		t.Fatal("identity lost")
	}
	if inspectorMessageIndex(msgs, InspectorTarget{Seq: 21}) != 1 || inspectorMessageIndex(nil, InspectorTarget{Seq: 20}) != -1 {
		t.Fatal("invalid fallback")
	}
	n := &InspectorNavigation{SessionID: "s", Target: &InspectorTarget{ID: "exact", Seq: 20}}
	expanded := map[string]bool{}
	m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s", Messages: msgs}, ThinkingExpanded: &expanded}
	controls := map[string]transcriptControl{"fold": {key: "fold", members: []string{"exact"}}}
	list := &ui.ListState{}
	prepareSessionInspector(m, msgs, controls, list)
	if !expanded["fold"] || n.Target == nil {
		t.Fatal("ancestor not unfolded")
	}
	controls["fold"] = transcriptControl{key: "fold", members: []string{"exact"}, open: true}
	prepareSessionInspector(m, msgs, controls, list)
	if n.Target != nil || n.Highlight != "exact" {
		t.Fatal("target did not land")
	}
	n.Target = &InspectorTarget{ID: "first"}
	m.Snapshot.SelectedID = "other"
	prepareSessionInspector(m, msgs, nil, list)
	if n.Highlight != "" || n.Target != nil || n.reveal.pending || !n.reveal.started.IsZero() {
		t.Fatal("cross-session gesture was not disposed")
	}
}
