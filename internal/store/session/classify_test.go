package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClassifyToolCall(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "tool/call",
		"seq": 1,
		"time": 0,
		"data": {"name": "bash", "arguments": "{\"command\":\"ls\"}", "callId": "c1", "step": 0, "turn": 0}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Kind != MessageKindTool || msg.EventType != "tool/call" {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
	if len(msg.Raw) == 0 {
		t.Fatal("missing raw payload")
	}
	if msg.Role != "tool" {
		t.Fatalf("role %q", msg.Role)
	}
	if msg.Args == "" || !strings.Contains(msg.Args, "command") {
		t.Fatalf("args %q", msg.Args)
	}
}

func TestClassifyThinking(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "assistant/thinking",
		"seq": 2,
		"data": {"text": "hmm"}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Kind != MessageKindThinking || msg.Body != "hmm" {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
}

func TestClassifyUserMessage(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "user/message",
		"seq": 3,
		"data": {"content": [{"type":"text","text":"hi"}]}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Kind != MessageKindText || msg.Role != "user" || msg.Body != "hi" {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
	if msg.Seq != 3 {
		t.Fatalf("seq = %v", msg.Seq)
	}
}

func TestClassifyCompaction(t *testing.T) {
	raw := json.RawMessage(`{"type":"compaction/summary","seq":11,"data":{"text":"trimmed"}}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Kind != MessageKindCompaction {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
}

func TestClassifyDeliverables(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "deliverables/presented",
		"seq": 9,
		"data": {"callId":"c","turn":0,"files":[{"path":"out.md","description":"notes"}]}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Kind != MessageKindDeliverables || len(msg.Files) != 1 {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
	if msg.Files[0] != "out.md — notes" {
		t.Fatalf("files = %#v", msg.Files)
	}
}

func TestClassifyToolWorkflow(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "tool-workflow/run-start",
		"seq": 4,
		"data": {"name": "wf", "runId": "r1"}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Kind != MessageKindTool {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
}

func TestClassifyTypedWebSearchPresentation(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "tool/result",
		"seq": 12,
		"data": {
			"message": {"source": {"kind": "tool", "callId": "c1"}, "toolCallId": "c1", "content": [{"type": "text", "text": "search output"}]},
			"meta": {"sources": [{"url": "https://example.com/a", "title": "Example", "snippet": "excerpt", "publishedAt": "2026-10-17"}], "answer": "summary", "truncated": false}
		}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Web == nil || msg.Web.Kind != "search" {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
	if len(msg.Web.Sources) != 1 || msg.Web.Sources[0].URL != "https://example.com/a" || msg.Web.Sources[0].Snippet != "excerpt" {
		t.Fatalf("web=%#v", msg.Web)
	}
	if msg.Web.Answer != "summary" || msg.Web.Truncated {
		t.Fatalf("web=%#v", msg.Web)
	}
}

func TestClassifyAssistantMessageID(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "assistant/message",
		"seq": 6,
		"data": {
			"message": {"id": "asst-99", "role": "assistant", "content": [{"type":"text","text":"ok"}]},
			"step": 0,
			"turn": 0,
			"stream": []
		}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.ID != "asst-99" || msg.Role != "assistant" {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
}

func TestClassifyIgnoresEmptyTurn(t *testing.T) {
	raw := json.RawMessage(`{"type": "turn/start", "seq": 5, "data": {"turn": 0}}`)
	if _, ok := classifyPersistenceEvent(raw); ok {
		t.Fatal("expected skip")
	}
}
