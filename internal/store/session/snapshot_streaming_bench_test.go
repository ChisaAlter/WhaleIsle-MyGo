package session

import (
	"fmt"
	"testing"
)

// streamingRow is the row applyAssistantChunkLocked keeps for a reply that is
// still arriving: no journal sequence, no raw record, and a body that grows.
func streamingRow() Message {
	return Message{ID: "stream-row", Role: "assistant", Kind: MessageKindText, Streaming: true, Body: "partial"}
}

func streamTick(s *Store) {
	s.mu.Lock()
	last := len(s.messages) - 1
	s.messages[last].Body += "x"
	s.invalidateViewLocked()
	s.mu.Unlock()
}

// BenchmarkSnapshotWhileStreaming is the per-frame cost of reading the view while
// the last row grows, against the same store read with nothing changing between
// frames. The ratio is what streaming costs on top of an idle frame.
func BenchmarkSnapshotWhileStreaming(b *testing.B) {
	for _, size := range []int{500, 2000, 8000} {
		b.Run(fmt.Sprintf("messages_%d_streaming_row", size), func(b *testing.B) {
			s := benchmarkStore(size)
			s.messages = append(s.messages, streamingRow())
			_ = s.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				streamTick(s)
				_ = s.Snapshot()
			}
		})
		// A real stream also has an active attempt, which makes the store re-run its
		// whole message projection on every read.
		b.Run(fmt.Sprintf("messages_%d_live_attempt", size), func(b *testing.B) {
			s := benchmarkStore(size)
			s.messages = append(s.messages, streamingRow())
			s.assistantStream.active = &assistantStreamAttempt{AttemptID: "bench", Turn: 1, Step: 1}
			_ = s.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				streamTick(s)
				_ = s.Snapshot()
			}
		})
		b.Run(fmt.Sprintf("messages_%d_frame_live_attempt", size), func(b *testing.B) {
			s := benchmarkStore(size)
			s.messages = append(s.messages, streamingRow())
			s.assistantStream.active = &assistantStreamAttempt{AttemptID: "bench", Turn: 1, Step: 1}
			frame := s.BeginFrame()
			defer frame.EndFrame()
			_ = frame.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				streamTick(s)
				_ = frame.Snapshot()
			}
		})
		b.Run(fmt.Sprintf("messages_%d_nothing_changed", size), func(b *testing.B) {
			s := benchmarkStore(size)
			_ = s.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.fire()
				_ = s.Snapshot()
			}
		})
	}
}
