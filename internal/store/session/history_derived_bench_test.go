package session

import (
	"encoding/json"
	"fmt"
	"testing"

	schedstore "whaleisle/internal/store/schedule"
)

// The loaded native-history fixture has repeated durable user/assistant/end and
// delivery rows, with a small number of schedule receipts among the whole source.
func historyDerivedLongFixture(rows int) ViewState {
	v := ViewState{SelectedID: "loaded-history", PlanHistoryBinding: "fixture/follow"}
	for i := range rows {
		typeName := "assistant/message"
		switch i % 4 {
		case 0:
			typeName = "user/message"
		case 3:
			typeName = "turn/end"
		}
		raw := json.RawMessage(fmt.Sprintf(`{"type":%q,"seq":%d,"data":{"turn":%d,"step":1,"usage":{"outputTokens":120},"message":{"id":"message-%d","content":[{"type":"text","text":"Synthetic answer with durable history and recorded accounting."}]}}}`, typeName, i+1, i/4+1, i+1))
		m := Message{Seq: float64(i + 1), Raw: raw}
		if typeName == "assistant/message" {
			m.TrajectoryTime = &TrajectoryTiming{FirstTokenAt: historyDerivedNumber(1000), CompletedAt: historyDerivedNumber(1600)}
		}
		v.Messages = append(v.Messages, m)
	}
	v.Messages = append(v.Messages, historyDerivedFixture().Messages[:5]...)
	return v
}

var historyDerivedPerfSink historyRecordedPerformance
var historyDerivedCardsSink map[float64][]schedstore.Entry

func BenchmarkHistoryDerived(b *testing.B) {
	for _, rows := range []int{400, 2000} {
		v := historyDerivedLongFixture(rows)
		b.Run(fmt.Sprintf("rows_%d/performance_oracle", rows), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				historyDerivedPerfSink.tokens, historyDerivedPerfSink.millis, historyDerivedPerfSink.steps = RecordedPerformance(v.Messages)
			}
		})
		b.Run(fmt.Sprintf("rows_%d/performance_warm", rows), func(b *testing.B) {
			var memo HistoryDerivedMemo
			memo.RecordedPerformance(v)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				historyDerivedPerfSink.tokens, historyDerivedPerfSink.millis, historyDerivedPerfSink.steps = memo.RecordedPerformance(v)
			}
		})
		b.Run(fmt.Sprintf("rows_%d/schedule_oracle", rows), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				historyDerivedCardsSink = ScheduleCards(v.Messages, v.SelectedID)
			}
		})
		b.Run(fmt.Sprintf("rows_%d/schedule_warm", rows), func(b *testing.B) {
			var memo HistoryDerivedMemo
			memo.ScheduleCards(v)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				historyDerivedCardsSink = memo.ScheduleCards(v)
			}
		})
		b.Run(fmt.Sprintf("rows_%d/combined_warm", rows), func(b *testing.B) {
			var memo HistoryDerivedMemo
			memo.RecordedPerformance(v)
			memo.ScheduleCards(v)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				memo.RecordedPerformance(v)
				historyDerivedCardsSink = memo.ScheduleCards(v)
			}
		})
	}
}
