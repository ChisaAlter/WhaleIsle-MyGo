package session

import (
	"encoding/json"
	"testing"
)

func TestExtractTextish(t *testing.T) {
	raw := json.RawMessage(`{"content":[{"type":"text","text":"hi"}]}`)
	if got := extractTextish(raw); got != "hi" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyAssistantStreamChunks(t *testing.T) {
	s := New()
	s.messages = nil
	start, _ := json.Marshal(map[string]any{
		"type": "assistant-stream",
		"frame": map[string]any{
			"type":            "start",
			"attemptId":       "a",
			"revision":        1,
			"turn":            0,
			"step":            0,
			"startedAfterSeq": -1,
		},
	})
	s.applyFollowFrame(start)
	chunk, _ := json.Marshal(map[string]any{
		"type": "assistant-stream",
		"frame": map[string]any{
			"type":      "chunk",
			"attemptId": "a",
			"revision":  2,
			"index":     0,
			"time":      0,
			"chunk":     map[string]any{"type": "text", "text": "Hel"},
		},
	})
	s.applyFollowFrame(chunk)
	chunk2, _ := json.Marshal(map[string]any{
		"type": "assistant-stream",
		"frame": map[string]any{
			"type":      "chunk",
			"attemptId": "a",
			"revision":  3,
			"index":     1,
			"time":      1,
			"chunk":     map[string]any{"type": "text", "text": "lo"},
		},
	})
	s.applyFollowFrame(chunk2)
	st := s.Snapshot()
	if len(st.Messages) != 1 || st.Messages[0].Body != "Hello" || !st.Messages[0].Streaming {
		t.Fatalf("%#v", st.Messages)
	}
}
