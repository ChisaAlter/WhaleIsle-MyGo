package session

import (
	"encoding/json"
	"fmt"
	"testing"
)

// benchmarkStore builds a synthetic history with the shape and size of a long
// real session: many step/assistant events that carry usage payloads, plus the
// trajectory boundaries the two expensive builders decode every frame.
func benchmarkStore(messages int) *Store {
	s := &Store{}
	s.selectedID = "bench"
	s.trajectoryBoundaries = map[float64]json.RawMessage{}
	for i := 0; i < messages; i++ {
		seq := float64(i + 1)
		turn := float64(i/12 + 1)
		step := float64(i%12 + 1)
		var raw string
		switch i % 3 {
		case 0:
			raw = fmt.Sprintf(`{"type":"step/start","seq":%d,"data":{"turn":%d,"step":%d}}`, int(seq), int(turn), int(step))
		case 1:
			raw = fmt.Sprintf(`{"type":"assistant/message","seq":%d,"data":{"turn":%d,"step":%d,"text":"chunk %d"}}`, int(seq), int(turn), int(step), i)
		default:
			raw = fmt.Sprintf(`{"type":"llm/usage","seq":%d,"data":{"turn":%d,"step":%d,"usage":{"inputTokens":%d,"outputTokens":%d}}}`, int(seq), int(turn), int(step), 900+i, 40+i)
		}
		if i%40 == 0 {
			s.trajectoryBoundaries[seq] = json.RawMessage(raw)
			continue
		}
		s.messages = append(s.messages, Message{ID: fmt.Sprintf("m%d", i), Role: "assistant", Seq: seq, Raw: json.RawMessage(raw), Body: "chunk"})
	}
	return s
}

func BenchmarkSnapshotLongHistory(b *testing.B) {
	for _, size := range []int{500, 2000, 8000} {
		// Uncached: fire() simulates one observable change between frames, so
		// every iteration pays the full projection rebuild.
		b.Run(fmt.Sprintf("messages_%d", size), func(b *testing.B) {
			s := benchmarkStore(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.fire()
				_ = s.Snapshot()
			}
		})
		// Cached reads pay for one clone, not for rebuilding the projection.
		// The priming Snapshot runs before the timer and no fire() follows it.
		b.Run(fmt.Sprintf("messages_%d_cached", size), func(b *testing.B) {
			s := benchmarkStore(size)
			_ = s.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.Snapshot()
			}
		})
		b.Run(fmt.Sprintf("messages_%d_frame_reuse", size), func(b *testing.B) {
			s := benchmarkStore(size)
			frame := s.BeginFrame()
			defer frame.EndFrame()
			_ = frame.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = frame.Snapshot()
			}
		})
		b.Run(fmt.Sprintf("messages_%d_new_frame", size), func(b *testing.B) {
			s := benchmarkStore(size)
			_ = s.Snapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				frame := s.BeginFrame()
				_ = frame.Snapshot()
				frame.EndFrame()
			}
		})
	}
}

// TestSnapshotParseCacheInvalidatesOnAppend guards the cache key: appending a
// new event must be visible without waiting for anything else to change.
func TestSnapshotParseCacheInvalidatesOnAppend(t *testing.T) {
	s := benchmarkStore(30)
	first := s.Snapshot()
	if len(first.TrajectoryRequests) == 0 {
		t.Fatal("expected trajectory requests from the fixture")
	}
	before := len(first.TrajectoryRequests)
	s.mu.Lock()
	s.messages = append(s.messages, Message{ID: "extra", Role: "assistant", Seq: 31, Raw: json.RawMessage(`{"type":"step/start","seq":31,"data":{"turn":99,"step":1}}`), Body: "x"})
	s.invalidateViewLocked()
	s.mu.Unlock()
	second := s.Snapshot()
	if len(second.TrajectoryRequests) <= before {
		t.Fatalf("append did not add a request: before=%d after=%d", before, len(second.TrajectoryRequests))
	}
	if second.TurnUsage == nil {
		t.Fatal("turn usage missing after append")
	}
}

// TestSnapshotCopyIsIndependent pins the contract the earlier value-cache broke:
// mutating a returned request must never corrupt what the next Snapshot sees.
func TestSnapshotCopyIsIndependent(t *testing.T) {
	s := benchmarkStore(30)
	snapshot := s.Snapshot()
	if len(snapshot.TrajectoryRequests) == 0 {
		t.Fatal("expected trajectory requests")
	}
	snapshot.TrajectoryRequests[0].Status = "mutated by caller"
	snapshot.TrajectoryRequests[0].Events = append(snapshot.TrajectoryRequests[0].Events, json.RawMessage(`{"type":"x"}`))
	again := s.Snapshot()
	if again.TrajectoryRequests[0].Status == "mutated by caller" {
		t.Fatal("caller mutation leaked into the cached snapshot")
	}
}
