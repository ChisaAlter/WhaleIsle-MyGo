package chat

import (
	"encoding/json"
	"fmt"
	"testing"

	"whaleisle/internal/store/session"
)

// Each turn has a human row, deliverable, short final answer and turn boundary.
// The 500-turn benchmark is exactly the 2000-row task40 audit shape.
func mentionLRUHistory(turns int) ([]session.Message, []session.Message) {
	messages := make([]session.Message, 0, turns*4)
	closings := make([]session.Message, 0, turns)
	for turn := 0; turn < turns; turn++ {
		seq := turn*4 + 1
		path := fmt.Sprintf("/synthetic/report-%03d.txt", turn)
		body := fmt.Sprintf("See report-%03d.txt.", turn)
		rows := []session.Message{
			{ID: fmt.Sprintf("u%d", turn), Seq: float64(seq), Kind: session.MessageKindText, Role: "user", Body: "Prepare the synthetic report.", Raw: json.RawMessage(fmt.Sprintf(`{"type":"user/message","seq":%d,"data":{"turn":%d,"content":[{"type":"text","text":"Prepare the synthetic report."}]}}`, seq, turn))},
			{ID: fmt.Sprintf("d%d", turn), Seq: float64(seq + 1), Kind: session.MessageKindDeliverables, Role: "assistant", EventType: "deliverables/presented", Raw: json.RawMessage(fmt.Sprintf(`{"type":"deliverables/presented","seq":%d,"data":{"turn":%d,"files":[{"path":%q}]}}`, seq+1, turn, path))},
			{ID: fmt.Sprintf("a%d", turn), SourceMessageID: fmt.Sprintf("a%d", turn), Seq: float64(seq + 2), Kind: session.MessageKindText, Role: "assistant", Body: body, Raw: json.RawMessage(fmt.Sprintf(`{"type":"assistant/message","seq":%d,"data":{"turn":%d,"step":1,"message":{"id":"a%d","content":[{"type":"text","text":%q}]}}}`, seq+2, turn, turn, body))},
			{ID: fmt.Sprintf("e%d", turn), Seq: float64(seq + 3), Kind: session.MessageKindText, Role: "status", EventType: "turn/end", Raw: json.RawMessage(fmt.Sprintf(`{"type":"turn/end","seq":%d,"data":{"turn":%d,"reason":{"kind":"completed"}}}`, seq+3, turn))},
		}
		messages = append(messages, rows...)
		closings = append(closings, rows[2])
	}
	return messages, closings
}

var mentionLRUResolverSink func(string) string

func BenchmarkMentionMemoLRUWarmRounds(b *testing.B) {
	messages, allClosings := mentionLRUHistory(500)
	for _, count := range []int{1, 2, 4, 8, 16} {
		closings := allClosings[len(allClosings)-count:]
		b.Run(fmt.Sprintf("closings_%d", count), func(b *testing.B) {
			memo := mentionMemoStore{}
			for _, closing := range closings {
				mentionLRUResolverSink = memo.resolver(messages, closing, buildMentionResolver)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for _, closing := range closings {
					mentionLRUResolverSink = memo.resolver(messages, closing, buildMentionResolver)
				}
			}
		})
	}
}
