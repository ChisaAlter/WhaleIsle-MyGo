package chat

import (
	"encoding/json"
	"fmt"
	"testing"

	"whaleisle/internal/store/session"
)

func deliveryMemoValue() *session.TurnDelivery {
	return &session.TurnDelivery{
		Turn:    1,
		Changes: &session.TurnFileTarget{Kind: "changes", Turn: 1, Seq: 2, Index: 3},
		Files:   []session.TurnPresentedFile{{Target: session.TurnFileTarget{Kind: "presented", Turn: 1, Seq: 4, Index: 5}, Path: "/synthetic/report-a.txt", Description: "report A"}},
	}
}

func deliveryMemoSnapshot() session.ViewState {
	return session.ViewState{SelectedID: "delivery-memo", PlanHistoryBinding: "synthetic", Messages: []session.Message{{ID: "delivery-card", Seq: 5, Kind: session.MessageKindTurnDelivery, Role: "assistant", Raw: json.RawMessage(`{}`), TurnDelivery: deliveryMemoValue()}}}
}

func deliveryMemoLongSnapshot(rows, cards int) session.ViewState {
	messages := make([]session.Message, rows)
	for index := range messages {
		seq, turn := index+1, index/4+1
		typeName, role := "assistant/message", "assistant"
		if index%4 == 0 {
			typeName, role = "user/message", "user"
		} else if index%4 == 3 {
			typeName, role = "turn/end", "status"
		}
		messages[index] = session.Message{ID: fmt.Sprintf("message-%d", seq), Seq: float64(seq), Kind: session.MessageKindText, Role: role, Body: "Synthetic report summary.", EventType: typeName, Raw: json.RawMessage(fmt.Sprintf(`{"type":%q,"seq":%d,"data":{"turn":%d,"step":1,"reason":{"kind":"completed"},"message":{"id":"message-%d","content":[{"type":"text","text":"Synthetic report summary."}]}}}`, typeName, seq, turn, seq))}
	}
	for index := 0; index < cards; index++ {
		row := len(messages) - 1 - index*4
		messages[row] = session.Message{ID: fmt.Sprintf("delivery-card-%d", index), Seq: messages[row].Seq, Kind: session.MessageKindTurnDelivery, Role: "assistant", TurnDelivery: deliveryMemoValue()}
	}
	return session.ViewState{SelectedID: fmt.Sprintf("delivery-memo-%d-%d", rows, cards), PlanHistoryBinding: "synthetic", Messages: messages}
}

var deliveryMemoProjectionSink []session.Message

func BenchmarkDeliveryProjectionMemo(b *testing.B) {
	for _, rows := range []int{2000, 8000} {
		for _, cards := range []int{0, 1, 3} {
			snap := deliveryMemoLongSnapshot(rows, cards)
			b.Run(fmt.Sprintf("rows_%d_cards_%d", rows, cards), func(b *testing.B) {
				projectChatMessages(snap)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					deliveryMemoProjectionSink = projectChatMessages(snap)
				}
			})
		}
	}
}
