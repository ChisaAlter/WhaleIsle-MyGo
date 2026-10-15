package session

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func toolLifecycleEvent(typ string, seq int, data string) Message {
	raw := json.RawMessage(fmt.Sprintf(`{"type":%q,"seq":%d,"time":%d,"data":%s}`, typ, seq, seq*10, data))
	return Message{ID: fmt.Sprint(seq), Kind: MessageKindTool, Role: "tool", EventType: typ, Seq: float64(seq), Raw: raw, TurnSeq: 1}
}

func TestToolLifecycleCallResultAndJournal(t *testing.T) {
	call := toolLifecycleEvent("tool/call", 2, `{"callId":"c","name":"web_search","arguments":"{\"query\":\"same\"}"}`)
	result := toolLifecycleEvent("tool/result", 5, `{"message":{"toolCallId":"c","source":{"callId":"c"},"content":[{"type":"text","text":"https://example.com"},{"type":"image","attachment":{"attachmentId":"image1"}}]}}`)
	user := Message{ID: "user", Kind: MessageKindText, Role: "user", Body: "same", Seq: 1, TurnSeq: 1, TurnOpening: true}
	other := toolLifecycleEvent("tool/call", 6, `{"callId":"other","name":"web_search","arguments":"{\"query\":\"same\"}"}`)
	messages := []Message{user, call, {ID: "assistant", Kind: MessageKindText, Role: "assistant", Body: "same", Streaming: true}, result, other}
	before, _ := json.Marshal(messages)
	rows := ProjectToolMessages(messages)
	if len(rows) != 4 || rows[1].Tool == nil || rows[1].Tool.State != "ok" || rows[1].Tool.Name != "web_search" || rows[1].ID != "tool:c" {
		t.Fatalf("rows: %#v", rows)
	}
	view := rows[1].Tool
	if view.Call == nil || view.Result == nil || len(view.Events) != 2 || view.Arguments != `{"query":"same"}` || !reflect.DeepEqual(view.Result.Raw, result.Raw) {
		t.Fatalf("lost inputs/results: %#v", view)
	}
	if !reflect.DeepEqual(rows[0], user) || rows[2].ID != "assistant" || rows[3].Tool.CallID != "other" {
		t.Fatal("non-tool/equal-text rows were swallowed")
	}
	after, _ := json.Marshal(messages)
	if string(before) != string(after) {
		t.Fatal("journal mutated")
	}
	view.Result.Raw[0] = 'x'
	view.Events[0].Raw[0] = 'x'
	if messages[3].Raw[0] != '{' || messages[1].Raw[0] != '{' {
		t.Fatal("projection aliases source raw")
	}
}

func TestToolLifecycleHistoryEnrichmentAndTerminalPriority(t *testing.T) {
	done := toolLifecycleEvent("tool/result", 20, `{"message":{"source":{"callId":"late"},"isError":true,"content":[]},"error":{"code":"interrupted","name":"AbortError","reason":"cancelled"}}`)
	rows := ProjectToolMessages([]Message{done})
	if len(rows) != 1 || rows[0].Tool.State != "stopped" || rows[0].Tool.Name != "tool" {
		t.Fatalf("result only: %#v", rows)
	}
	start := toolLifecycleEvent("tool/call", 10, `{"callId":"late","name":"bash","arguments":"echo x"}`)
	lateReplay := toolLifecycleEvent("tool/call", 30, `{"callId":"late","name":"bash","arguments":"echo changed"}`)
	rows = ProjectToolMessages([]Message{start, done, lateReplay})
	if len(rows) != 1 || rows[0].Tool.State != "stopped" || rows[0].Tool.Name != "bash" || rows[0].Tool.Arguments != "echo x" || rows[0].ID != "tool:late" {
		t.Fatalf("enrichment: %#v", rows)
	}
}

func TestToolLifecycleNestedPTCAndMissingParents(t *testing.T) {
	child := toolLifecycleEvent("tool/ptc-dispatch-start", 3, `{"subCallId":"child","parentCallId":"root","rootCallId":"root","name":"read","arguments":{"path":"a"}}`)
	grand := toolLifecycleEvent("tool/ptc-dispatch", 4, `{"subCallId":"grand","parentCallId":"child","rootCallId":"root","name":"bash","arguments":{"command":"bad"},"isError":true,"content":[],"error":{"name":"Failure","code":"bad","reason":"failed"}}`)
	root := toolLifecycleEvent("tool/call", 2, `{"callId":"root","name":"run_code","arguments":"code"}`)
	rows := ProjectToolMessages([]Message{child, grand})
	if len(rows) != 1 || rows[0].Tool.ParentIssue != "missing parent" || len(rows[0].Tool.Children) != 1 {
		t.Fatalf("orphan: %#v", rows)
	}
	rows = ProjectToolMessages([]Message{root, child, grand})
	r := rows[0].Tool
	if len(rows) != 1 || len(r.Children) != 1 || len(r.Children[0].Children) != 1 || r.Children[0].Children[0].State != "error" || r.Children[0].Children[0].Arguments != `{"command":"bad"}` {
		t.Fatalf("tree: %#v", r)
	}
}

func TestToolLifecycleMalformedRelationsAndUnknownEvents(t *testing.T) {
	a := toolLifecycleEvent("tool/ptc-dispatch-start", 1, `{"subCallId":"a","parentCallId":"b","name":"read"}`)
	b := toolLifecycleEvent("tool/ptc-dispatch-start", 2, `{"subCallId":"b","parentCallId":"a","name":"read"}`)
	self := toolLifecycleEvent("tool/ptc-dispatch-start", 3, `{"subCallId":"self","parentCallId":"self","name":"read"}`)
	missing := toolLifecycleEvent("tool/result", 4, `{"message":{"content":[]}}`)
	conflict := toolLifecycleEvent("tool/result", 5, `{"message":{"toolCallId":"a","source":{"callId":"b"}}}`)
	unknown := toolLifecycleEvent("tool-workflow/run-start", 6, `{"runId":"a","name":"read"}`)
	rows := ProjectToolMessages([]Message{a, b, self, missing, conflict, unknown})
	if len(rows) != 5 {
		t.Fatalf("lost rows: %#v", rows)
	}
	seen := map[string]bool{}
	stack := []*ToolCallView{rows[0].Tool, rows[1].Tool}
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if v == nil || seen[v.CallID] {
			t.Fatal("cycle or duplicate")
		}
		seen[v.CallID] = true
		stack = append(stack, v.Children...)
	}
	if len(seen) != 3 || rows[2].ID != missing.ID || rows[3].ID != conflict.ID || rows[4].ID != unknown.ID {
		t.Fatal("malformed events disappeared")
	}
}

func TestToolLifecycleDeepTreeDoesNotRecurse(t *testing.T) {
	const depth = 3000
	events := make([]Message, 0, depth)
	for i := 0; i < depth; i++ {
		parent := ""
		if i > 0 {
			parent = fmt.Sprint(i - 1)
		}
		events = append(events, toolLifecycleEvent("tool/ptc-dispatch-start", i+1, fmt.Sprintf(`{"subCallId":%q,"parentCallId":%q,"name":"read"}`, fmt.Sprint(i), parent)))
	}
	rows := ProjectToolMessages(events)
	if len(rows) != 1 {
		t.Fatalf("roots=%d", len(rows))
	}
	count := 0
	for v := rows[0].Tool; v != nil; {
		count++
		if len(v.Children) == 0 {
			break
		}
		v = v.Children[0]
	}
	if count != depth {
		t.Fatalf("depth=%d", count)
	}
}
