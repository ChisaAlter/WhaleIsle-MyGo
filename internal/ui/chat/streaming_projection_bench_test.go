package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

func buildStreamingChatMessages(n int) []session.Message {
	var messages []session.Message
	turn := 0
	for i := 1; i <= n; i++ {
		switch i % 4 {
		case 1:
			turn++
			messages = append(messages, session.Message{
				ID:        fmt.Sprintf("user-%d", i),
				Seq:       float64(i * 10),
				TurnSeq:   float64(turn * 100),
				Kind:      session.MessageKindText,
				Role:      "user",
				EventType: "user/message",
				Body:      fmt.Sprintf("User prompt %d", i),
				Raw:       json.RawMessage(fmt.Sprintf(`{"type":"user/message","seq":%d,"time":%d,"data":{"turn":%d,"content":[{"type":"text","text":"User prompt %d"}]}}`, i*10, 1700000000000+i*100, turn, i)),
			})
		case 2:
			callID := fmt.Sprintf("call-%d", i)
			messages = append(messages,
				session.Message{
					ID:        fmt.Sprintf("tool-call-%d", i),
					Seq:       float64(i*10 + 1),
					TurnSeq:   float64(turn * 100),
					Kind:      session.MessageKindTool,
					Role:      "tool",
					EventType: "tool/call",
					Body:      "read_file",
					Raw:       json.RawMessage(fmt.Sprintf(`{"type":"tool/call","seq":%d,"time":%d,"data":{"turn":%d,"step":1,"callId":%q,"name":"read_file","arguments":"{\"path\":\"internal/store/session/store.go\",\"startLine\":1,\"endLine\":200}"}}`, i*10+1, 1700000000000+i*100+1, turn, callID)),
				},
				session.Message{
					ID:        fmt.Sprintf("tool-res-%d", i),
					Seq:       float64(i*10 + 2),
					TurnSeq:   float64(turn * 100),
					Kind:      session.MessageKindTool,
					Role:      "tool",
					EventType: "tool/result",
					Body:      "package session",
					Raw:       json.RawMessage(fmt.Sprintf(`{"type":"tool/result","seq":%d,"time":%d,"data":{"turn":%d,"step":1,"message":{"toolCallId":%q,"isError":false,"content":[{"type":"text","text":"package session\n// file contents here\n"}]}}}`, i*10+2, 1700000000000+i*100+2, turn, callID)),
				},
			)
		case 3:
			messages = append(messages, session.Message{
				ID:        fmt.Sprintf("asst-%d", i),
				Seq:       float64(i * 10),
				TurnSeq:   float64(turn * 100),
				Kind:      session.MessageKindText,
				Role:      "assistant",
				EventType: "assistant/message",
				Body:      fmt.Sprintf("Assistant response for item %d with markdown details.", i),
				Raw:       json.RawMessage(fmt.Sprintf(`{"type":"assistant/message","seq":%d,"time":%d,"data":{"turn":%d,"step":2,"usage":{"inputTokens":1200,"outputTokens":350},"message":{"id":"msg-%d","source":{"provider":"openai","model":"gpt-5"},"content":[{"type":"text","text":"Assistant response for item %d with markdown details."}]}}}`, i*10, 1700000000000+i*100, turn, i, i)),
			})
		case 0:
			messages = append(messages, session.Message{
				ID:        fmt.Sprintf("turn-end-%d", i),
				Seq:       float64(i * 10),
				TurnSeq:   float64(turn * 100),
				Kind:      session.MessageKindText,
				Role:      "status",
				EventType: "turn/end",
				Body:      "120",
				Raw:       json.RawMessage(fmt.Sprintf(`{"type":"turn/end","seq":%d,"time":%d,"data":{"turn":%d}}`, i*10, 1700000000000+i*100, turn)),
			})
		}
	}
	liveTurn := float64(turn + 1)
	liveStep := float64(1)
	messages = append(messages, session.Message{
		ID:               "live-stream-id",
		TurnSeq:          liveTurn * 100,
		Kind:             session.MessageKindText,
		Role:             "assistant",
		Streaming:        true,
		ConversationTurn: &liveTurn,
		ConversationStep: &liveStep,
		Body:             "Streaming reply: ",
	})
	return messages
}

func BenchmarkChatStreamingFrame(b *testing.B) {
	messages := buildStreamingChatMessages(2000)
	var viewMemo transcriptViewMemo
	var derived session.HistoryDerivedMemo
	expanded := map[string]bool{}
	var sb strings.Builder
	sb.WriteString("Streaming reply: ")
	// Warm up memos once
	warmSnap := session.New().Snapshot()
	warmSnap.SelectedID = "bench-session"
	warmSnap.PlanHistoryBinding = "bench-binding"
	warmSnap.Messages = messages
	_ = projectChatMessages(warmSnap)
	_, _, _, _ = viewMemo.resolve(Model{Snapshot: warmSnap, ThinkingExpanded: &expanded})
	_, _, _ = derived.RecordedPerformance(warmSnap)
	_ = derived.ScheduleCards(warmSnap)
	_ = projectedTodoBaselines(warmSnap)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.WriteString("chunk ")
		messages[len(messages)-1].Body = sb.String()
		snap := session.New().Snapshot()
		snap.SelectedID = "bench-session"
		snap.PlanHistoryBinding = "bench-binding"
		snap.Messages = messages
		m := Model{
			Snapshot:         snap,
			ThinkingExpanded: &expanded,
		}
		_ = projectChatMessages(snap)
		msgs, _, _, _ := viewMemo.resolve(m)
		_, _, _ = derived.RecordedPerformance(snap)
		_ = derived.ScheduleCards(snap)
		_ = projectedTodoBaselines(snap)
		if len(msgs) == 0 {
			b.Fatal("empty messages")
		}
	}
}

func BenchmarkChatStreamingThinkingFrame(b *testing.B) {
	messages := buildStreamingChatMessages(2000)
	messages[len(messages)-1].Kind = session.MessageKindThinking
	messages[len(messages)-1].Body = "Thinking step: "
	var viewMemo transcriptViewMemo
	var derived session.HistoryDerivedMemo
	expanded := map[string]bool{}
	var sb strings.Builder
	sb.WriteString("Thinking step: ")
	warmSnap := session.New().Snapshot()
	warmSnap.SelectedID = "bench-session-thinking"
	warmSnap.PlanHistoryBinding = "bench-binding"
	warmSnap.Messages = messages
	_ = projectChatMessages(warmSnap)
	_, _, _, _ = viewMemo.resolve(Model{Snapshot: warmSnap, ThinkingExpanded: &expanded})
	_, _, _ = derived.RecordedPerformance(warmSnap)
	_ = derived.ScheduleCards(warmSnap)
	_ = projectedTodoBaselines(warmSnap)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.WriteString("thought ")
		messages[len(messages)-1].Body = sb.String()
		snap := session.New().Snapshot()
		snap.SelectedID = "bench-session-thinking"
		snap.PlanHistoryBinding = "bench-binding"
		snap.Messages = messages
		m := Model{
			Snapshot:         snap,
			ThinkingExpanded: &expanded,
		}
		_ = projectChatMessages(snap)
		msgs, _, _, _ := viewMemo.resolve(m)
		_, _, _ = derived.RecordedPerformance(snap)
		_ = derived.ScheduleCards(snap)
		_ = projectedTodoBaselines(snap)
		if len(msgs) == 0 {
			b.Fatal("empty messages")
		}
	}
}

func BenchmarkChatStreamingToolCallFrame(b *testing.B) {
	messages := buildStreamingChatMessages(2000)
	tailIdx := len(messages) - 1
	messages[tailIdx] = session.Message{
		ID:        "tool:live-call-1",
		TurnSeq:   messages[tailIdx].TurnSeq,
		Kind:      session.MessageKindTool,
		Role:      "tool",
		Body:      "read_file",
		Args:      `{"path":"internal/`,
		Streaming: true,
		Tool: &session.ToolCallView{
			CallID:    "live-call-1",
			Name:      "read_file",
			Arguments: `{"path":"internal/`,
			State:     "preparing",
		},
	}
	var viewMemo transcriptViewMemo
	var derived session.HistoryDerivedMemo
	expanded := map[string]bool{}
	var sb strings.Builder
	sb.WriteString(`{"path":"internal/`)
	warmSnap := session.New().Snapshot()
	warmSnap.SelectedID = "bench-session-tool"
	warmSnap.PlanHistoryBinding = "bench-binding"
	warmSnap.Messages = messages
	_ = projectChatMessages(warmSnap)
	_, _, _, _ = viewMemo.resolve(Model{Snapshot: warmSnap, ThinkingExpanded: &expanded})
	_, _, _ = derived.RecordedPerformance(warmSnap)
	_ = derived.ScheduleCards(warmSnap)
	_ = projectedTodoBaselines(warmSnap)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.WriteString("sub/")
		args := sb.String()
		messages[tailIdx].Args = args
		messages[tailIdx].Tool = &session.ToolCallView{
			CallID:    "live-call-1",
			Name:      "read_file",
			Arguments: args,
			State:     "preparing",
		}
		snap := session.New().Snapshot()
		snap.SelectedID = "bench-session-tool"
		snap.PlanHistoryBinding = "bench-binding"
		snap.Messages = messages
		m := Model{
			Snapshot:         snap,
			ThinkingExpanded: &expanded,
		}
		_ = projectChatMessages(snap)
		msgs, _, _, _ := viewMemo.resolve(m)
		_, _, _ = derived.RecordedPerformance(snap)
		_ = derived.ScheduleCards(snap)
		_ = projectedTodoBaselines(snap)
		if len(msgs) == 0 {
			b.Fatal("empty messages")
		}
	}
}
