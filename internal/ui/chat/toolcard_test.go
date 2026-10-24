package chat

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/store/session"

	"whaleisle/internal/dsh/events"
)

func TestParseToolCallCard(t *testing.T) {
	raw, _ := json.Marshal(events.ToolCall{
		Type: events.TypeToolCall,
		Data: events.ToolCallData{Name: "read", Arguments: `{"path":"/tmp/x"}`},
	})
	info := ParseToolCard(events.TypeToolCall, raw)
	if info.Name != "read" || info.Status != "call" {
		t.Fatalf("%#v", info)
	}
	if info.Detail == "" {
		t.Fatal("expected args detail")
	}
}

func TestParseToolResultError(t *testing.T) {
	isErr := true
	raw, _ := json.Marshal(events.ToolResult{
		Type: events.TypeToolResult,
		Data: events.ToolResultData{
			Message: events.ToolResultDataMessage{
				Role:       "tool",
				ToolCallId: "c1",
				IsError:    &isErr,
				Content:    []json.RawMessage{json.RawMessage(`{"type":"text","text":"failed"}`)},
			},
		},
	})
	info := ParseToolCard(events.TypeToolResult, raw)
	if info.Status != "error" || !contains(info.Detail, "failed") {
		t.Fatalf("%#v", info)
	}
}

func TestToolResultOutputKeepsVerbatimBlocksAndEmptyErrorFallback(t *testing.T) {
	for _, tc := range []struct{ name, content, want string }{
		{"whitespace", `[{"type":"text","text":"  first\nsecond\n"}]`, "  first\nsecond\n"},
		{"empty text", `[{"type":"text","text":""}]`, ""},
		{"nontext evidence", `[{"type":"extension","text":"caption","key":true}]`, "{\n  \"type\": \"extension\",\n  \"text\": \"caption\",\n  \"key\": true\n}"},
		{"empty failed result", `[]`, "ReadError: ENOENT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := json.RawMessage(`{"type":"tool/result","data":{"error":{"name":"ReadError","code":"ENOENT","reason":"No duplicate reason"},"message":{"content":` + tc.content + `}}}`)
			if got := ParseToolCard(events.TypeToolResult, raw).Output; got != tc.want {
				t.Fatalf("standalone output=%q want%q", got, tc.want)
			}
			call := &session.ToolCallView{State: "error", Result: &session.Message{EventType: events.TypeToolResult, Raw: raw}}
			if got := toolLifecycleOutput(call); got != tc.want {
				t.Fatalf("paired output=%q want%q", got, tc.want)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
