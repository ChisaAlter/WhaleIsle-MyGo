package session

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh"
)

func TestOpenUIOwnerTicket(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "session"
	s.planHistoryBinding = "binding"
	s.messages = []Message{{ID: "message", Role: "assistant", Body: "```openui\nmain=Text(\"hello\")\n```"}}
	target, valid, err := s.CaptureOpenUI("binding", "session", "message", "main=Text(\"hello\")")
	if err != nil || !valid() || target.mode != PromptModeQueue {
		t.Fatal("valid owner", err)
	}
	if _, _, err = s.CaptureOpenUI("other", "session", "message", "main=Text(\"hello\")"); err == nil {
		t.Fatal("foreign host admitted")
	}
	s.selectedID = "other"
	if valid() {
		t.Fatal("navigation did not revoke")
	}
	s.selectedID = "session"
	s.messages = nil
	if valid() {
		t.Fatal("deleted message did not revoke")
	}
}
func TestWhaleUIToolOwnerStrictMetadata(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "s"
	s.planHistoryBinding = "b"
	call := Message{ID: "call", Kind: MessageKindTool, Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"c","name":"functions.render_ui"}}`)}
	result := Message{ID: "result", Kind: MessageKindTool, Seq: 8, Raw: json.RawMessage(`{"type":"tool/result","data":{"message":{"toolCallId":"c"},"meta":{"whaleUi":{"version":1,"language":"whale-ui","program":"root = Col([Text(\"Tool UI\")])"}}}}`)}
	s.messages = []Message{call, result}
	doc, ok := WhaleUIToolDocument(result)
	if !ok || doc.CallID != "c" {
		t.Fatal("valid tool doc not parsed")
	}
	target, valid, err := s.CaptureOpenUI("b", "s", "tool:c", doc.Program)
	if err != nil || !valid() || target.mode != PromptModeQueue {
		t.Fatal("tool owner not captured", err)
	}
	s.messages[0].Raw = json.RawMessage(`{"type":"tool/call","data":{"callId":"c","name":"other"}}`)
	if valid() {
		t.Fatal("wrong tool kept authority")
	}
	s.messages[0] = call
	s.messages[1].Raw = json.RawMessage(`{"type":"tool/result","data":{"message":{"toolCallId":"foreign"},"meta":{"whaleUi":{"version":1,"language":"whale-ui","program":"root = Col([Text(\"Tool UI\")])"}}}}`)
	if valid() {
		t.Fatal("call change kept authority")
	}
	for _, raw := range []string{
		`{"type":"tool/result","data":{"message":{"toolCallId":"c","isError":true},"meta":{"whaleUi":{"version":1,"language":"whale-ui","program":"root = Col([])"}}}}`,
		`{"type":"tool/result","data":{"message":{"toolCallId":"c","source":{"callId":"x"}},"meta":{"whaleUi":{"version":1,"language":"whale-ui","program":"root = Col([])"}}}}`,
		`{"type":"tool/result","data":{"message":{"toolCallId":"c"},"meta":{"whaleUi":{"version":2,"language":"whale-ui","program":"root = Col([])"}}}}`,
		`{"type":"tool/result","data":{"message":{"toolCallId":"c"},"content":"root = Col([])"}}`,
	} {
		if _, ok := WhaleUIToolDocument(Message{Kind: MessageKindTool, Raw: json.RawMessage(raw)}); ok {
			t.Fatal("invalid metadata admitted", raw)
		}
	}
}
func TestWhaleUIPanelOwnerGeneration(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "s"
	s.planHistoryBinding = "b"
	generation := 1
	_, valid, err := s.CaptureWhaleUIPanel("b", "s", func() bool { return generation == 1 })
	if err != nil || !valid() {
		t.Fatal(err)
	}
	generation++
	if valid() {
		t.Fatal("changed panel retained authority")
	}
}
