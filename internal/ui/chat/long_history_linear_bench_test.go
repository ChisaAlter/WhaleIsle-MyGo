package chat

import (
	"encoding/json"
	"fmt"
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
)

// Every completed turn has a process row, a final answer on another step and a
// boundary. Source IDs exercise the same lifecycle aggregation as real projected
// Assistant blocks. No persistence, Host, native window or model is involved.
func longHistoryLinearFixture(turns int) []session.Message {
	messages := make([]session.Message, 0, turns*4)
	for turn := 0; turn < turns; turn++ {
		base := turn*4 + 1
		coordinates := func(step int) json.RawMessage {
			return json.RawMessage(fmt.Sprintf(`{"data":{"turn":%d,"step":%d}}`, turn, step))
		}
		messages = append(messages,
			session.Message{ID: fmt.Sprintf("u%d", turn), Seq: float64(base), Role: "user", Kind: session.MessageKindText, Body: "Synthetic question", Raw: coordinates(0)},
			session.Message{ID: fmt.Sprintf("p%d", turn), Seq: float64(base + 1), Role: "thinking", Kind: session.MessageKindThinking, Body: "Synthetic process", Raw: coordinates(0)},
			session.Message{ID: fmt.Sprintf("a%d", turn), SourceMessageID: fmt.Sprintf("a%d", turn), Seq: float64(base + 2), Role: "assistant", Kind: session.MessageKindText, Body: "Synthetic final answer", Raw: coordinates(1)},
			session.Message{ID: fmt.Sprintf("e%d", turn), Seq: float64(base + 3), Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Raw: coordinates(1)},
		)
	}
	return messages
}

var longHistoryTranscriptSink []session.Message
var longHistoryControlsSink map[string]transcriptControl
var longHistoryActionsSink map[int]assistantActionTarget

func BenchmarkLongHistoryLinear(b *testing.B) {
	for _, turns := range []int{125, 500, 2000} {
		messages := longHistoryLinearFixture(turns)
		model := Model{Snapshot: session.ViewState{SelectedID: "synthetic-long-history", PlanHistoryBinding: "synthetic-host"}, Preferences: settings.ChatPreferences{TranscriptView: "standard"}}
		b.Run(fmt.Sprintf("transcript_rows_%d", len(messages)), func(b *testing.B) {
			_, _ = projectTranscript(messages, model) // Warm coordinate decoding memo.
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				longHistoryTranscriptSink, longHistoryControlsSink = projectTranscript(messages, model)
			}
		})
		b.Run(fmt.Sprintf("actions_rows_%d", len(messages)), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				longHistoryActionsSink = assistantActionTargets(messages)
			}
		})
	}
}
