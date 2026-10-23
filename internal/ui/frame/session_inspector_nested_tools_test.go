package frame

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

func TestInspectorNestedCallReverseKeepsOriginalOwnerRules(t *testing.T) {
	root := session.Message{ID: "source-root", Kind: session.MessageKindTool, Seq: 10, EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","seq":10,"data":{"turn":0,"step":0,"callId":"opaque-root","name":"functions.read","arguments":"{}"}}`)}
	child := session.Message{ID: "source-child", Kind: session.MessageKindTool, Seq: 11, EventType: "tool/ptc-dispatch-start", Raw: json.RawMessage(`{"type":"tool/ptc-dispatch-start","seq":11,"data":{"rootCallId":"opaque-root","parentCallId":"opaque-root","subCallId":"unrelated-child","name":"functions.read","arguments":{}}}`)}
	projected := session.ProjectToolMessages([]session.Message{root, child})
	p := SessionInspectorPaneState{Mode: "session-log", navigation: chat.InspectorNavigation{Nodes: projected}}
	target := projected[0].InspectorTarget()
	target.CallID = "unrelated-child"
	rows := []session.InspectorRecord{
		{Key: "event:10", Seq: 10, Type: root.EventType, Raw: root.Raw, ChatTarget: root.InspectorTarget()},
		{Key: "event:11", Seq: 11, Type: child.EventType, Raw: child.Raw, ChatTarget: child.InspectorTarget()},
	}
	if got := p.inspectorPickRecord(rows, target); got != "event:10" {
		t.Fatalf("nested dispatch changed original root-anchor reverse rule: %q", got)
	}
	if enriched := p.inspectorLogPickTarget(target); enriched.CallID != "unrelated-child" || enriched.AnchorSeq == nil || *enriched.AnchorSeq != 10 {
		t.Fatalf("root placement discarded particular call: %+v", enriched)
	}
	// A genuine tool/call for that identity outranks its containing node's
	// anchor. PTC start alone does not, matching the original log matcher.
	exact := session.Message{Seq: 15, EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","seq":15,"data":{"callId":"unrelated-child"}}`)}
	rows = append(rows, session.InspectorRecord{Key: "event:15", Seq: 15, Type: exact.EventType, Raw: exact.Raw, ChatTarget: exact.InspectorTarget()})
	if got := p.inspectorPickRecord(rows, target); got != "event:15" {
		t.Fatalf("exact real call lost priority: %q", got)
	}
	p.Mode = "chat-node"
	chatRows := []session.InspectorRecord{{Key: "chat:" + projected[0].ID, ChatTarget: projected[0].InspectorTarget()}}
	if got := p.inspectorPickRecord(chatRows, target); got != "chat:tool:opaque-root" {
		t.Fatalf("nested call invented a separate Chat node: %q", got)
	}
}
