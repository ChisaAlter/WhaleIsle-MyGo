package frame

import (
	"encoding/json"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestSessionInspectorFilterAndPaneIsolation(t *testing.T) {
	rows := []session.InspectorRecord{{Key: "turn", Type: "turn/start"}, {Key: "event", Parent: "turn", Type: "assistant/message"}, {Key: "chunk", Parent: "event", Type: "text-delta"}}
	got := inspectorVisibleRows(rows, "*DELTA*", nil)
	if len(got) != 3 {
		t.Fatal("filter lost ancestor context")
	}
	if len(inspectorVisibleRows(rows, "", map[string]inspectorDisclosure{"turn": {collapsed: true}})) != 1 {
		t.Fatal("fold lost descendants")
	}
	old := &WorkspacePane{Kind: "session-inspector-log", Inspector: &SessionInspectorPaneState{Mode: "chat-node", Filter: "tool", binding: "private-host", selected: "private-row", draft: "private-draft"}}
	added := &WorkspacePane{Kind: old.Kind}
	copySessionInspectorPane(added, old)
	added.Inspector.Filter = "new"
	if old.Inspector.Filter != "tool" || added.Inspector.selected != "" {
		t.Fatal("split state aliases")
	}
	raw, _ := json.Marshal(old)
	if strings.Contains(string(raw), "private") {
		t.Fatal("private source persisted")
	}
	var restored WorkspacePane
	if json.Unmarshal(raw, &restored) != nil || restored.Inspector.Mode != "chat-node" {
		t.Fatal("restore lost presentation")
	}
	if !independentPaneKind(old.Kind) {
		t.Fatal("not independent")
	}
}
