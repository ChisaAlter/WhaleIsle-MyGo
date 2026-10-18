package frame

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

func TestInspectorObjectsAndReferenceNavigation(t *testing.T) {
	source := session.InspectorSnapshot{Records: []session.InspectorRecord{
		{Key: "event:1", Seq: 1, Type: "turn/start", Raw: json.RawMessage(`{"type":"turn/start","data":{"turn":0}}`)},
		{Key: "event:2", Seq: 2, Type: "step/start", Raw: json.RawMessage(`{"type":"step/start","data":{"turn":0,"step":0}}`)},
		{Key: "event:3", Seq: 3, Type: "assistant/message", Raw: json.RawMessage(`{"type":"assistant/message","data":{"text":"fixture"}}`)},
	}, Chat: []session.InspectorRecord{{Key: "chat:actual", Seq: 3, Type: "text", Raw: json.RawMessage(`{"ID":"actual"}`)}}}
	objects := inspectorObjects(source)
	if len(objects["chat:actual"].Links) != 3 || string(objects["event:3/data"].Raw) != `{"text":"fixture"}` {
		t.Fatal("object relations missing")
	}
	p := &SessionInspectorPaneState{Filter: "excluded", breadcrumb: []string{"chat:actual"}}
	p.followInspectorReference("chat:actual", source.Chat)
	if p.Filter != "" || len(p.breadcrumb) != 2 || p.navigation.Target.ID != "actual" {
		t.Fatal("reference reveal failed")
	}
	if inspectorPickRecord(source.Chat, chat.InspectorTarget{ID: "actual", Seq: 3}) != "chat:actual" || inspectorPickRecord(source.Records, chat.InspectorTarget{ID: "actual", Seq: 3}) != "event:3" {
		t.Fatal("pick identity mismatch")
	}
}
func TestInspectorHiddenPaneCancelsPicking(t *testing.T) {
	s := session.New()
	p := &SessionInspectorPaneState{binding: s.InspectorBinding(), navigation: chat.InspectorNavigation{Picking: true, Target: &chat.InspectorTarget{ID: "old"}}}
	w := &WorkspacePanes{Root: &WorkspacePane{ID: "p", Kind: "session-inspector-log", Inspector: p}, Active: "p"}
	m := Model{Store: s, Sizes: &LayoutSizes{Workspace: w}, RightPaneOpen: false}
	if sessionInspectorChatNavigation(m) != nil || p.navigation.Picking || p.navigation.Target != nil {
		t.Fatal("hidden retained gesture")
	}
	m.RightPaneOpen = true
	if sessionInspectorChatNavigation(m) != &p.navigation {
		t.Fatal("visible not attached")
	}
	p.navigation.Picking = true
	m.DisableDeveloperTools = true
	if sessionInspectorChatNavigation(m) != nil || p.navigation.Picking {
		t.Fatal("disable retained gesture")
	}
}
