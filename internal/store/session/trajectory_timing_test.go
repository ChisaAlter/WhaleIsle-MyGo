package session

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh/api"
)

func trajectoryFixtureEvent(seq int, typ string, ms any, data map[string]any) json.RawMessage {
	event := map[string]any{"seq": seq, "type": typ, "data": data}
	if ms != nil {
		event["time"] = ms
	}
	raw, _ := json.Marshal(event)
	return raw
}

func trajectoryFixtureAssistant(seq int, ms any, turn, step int, stream []any) json.RawMessage {
	return trajectoryFixtureEvent(seq, "assistant/message", ms, map[string]any{"turn": turn, "step": step, "stream": stream, "message": map[string]any{"id": "answer", "role": "assistant", "source": map[string]any{"model": "fixture"}, "content": []any{map[string]any{"type": "text", "text": "answer"}}}})
}

func trajectoryFixtureFind(t *testing.T, s *Store, op string) *TrajectoryTiming {
	t.Helper()
	for _, m := range s.Snapshot().Messages {
		if m.TrajectoryTime != nil && m.TrajectoryTime.OperationID == op {
			return m.TrajectoryTime
		}
	}
	t.Fatalf("missing timing %s in %+v", op, s.Snapshot().Messages)
	return nil
}

func TestTrajectoryRecordedDurationsAndRetryTokens(t *testing.T) {
	s := New()
	apply := s.applyPersistenceEvent
	apply(trajectoryFixtureEvent(1, "step/start", 1000, map[string]any{"turn": 1, "step": 1}))
	apply(trajectoryFixtureEvent(2, "assistant/attempt", 1300, map[string]any{"turn": 1, "step": 1, "stream": []any{map[string]any{"type": "reasoning-chunks", "time0": 1100, "index": 0, "dt": []int{50}, "texts": []string{"", "thinking"}}}}))
	apply(trajectoryFixtureAssistant(3, 2000, 1, 1, []any{map[string]any{"type": "text-chunks", "time0": 1600, "index": 0, "dt": []int{}, "texts": []string{"answer"}}}))
	timing := trajectoryFixtureFind(t, s, "assistant:1:1")
	if duration, ok := timing.DurationMillis(); !ok || duration != 1000 {
		t.Fatalf("duration=%v known=%v timing=%+v", duration, ok, timing)
	}
	if timing.FirstTokenAt == nil || *timing.FirstTokenAt != 1150 {
		t.Fatalf("first token lost across retry: %+v", timing)
	}
	apply(trajectoryFixtureEvent(4, "tool/call", 2100, map[string]any{"callId": "root", "name": "fixture", "arguments": "{}"}))
	apply(trajectoryFixtureEvent(5, "tool/ptc-dispatch-start", 2200, map[string]any{"subCallId": "child", "parentCallId": "root", "name": "fixture", "arguments": map[string]any{}}))
	apply(trajectoryFixtureEvent(6, "tool/ptc-dispatch", 2500, map[string]any{"subCallId": "child", "parentCallId": "root", "name": "fixture", "content": []any{}}))
	apply(trajectoryFixtureEvent(7, "tool/result", 3000, map[string]any{"message": map[string]any{"source": map[string]any{"callId": "root"}, "content": []any{map[string]any{"type": "text", "text": "ok"}}}}))
	for id, want := range map[string]float64{"tool:root": 900, "tool:child": 300} {
		if got, ok := trajectoryFixtureFind(t, s, id).DurationMillis(); !ok || got != want {
			t.Fatalf("%s=%v known=%v", id, got, ok)
		}
	}
	apply(trajectoryFixtureEvent(8, "compaction/start", 3200, map[string]any{"compactionId": "c"}))
	apply(trajectoryFixtureEvent(9, "compaction/end", 3500, map[string]any{"compactionId": "c"}))
	if got, ok := trajectoryFixtureFind(t, s, "compaction:c").DurationMillis(); !ok || got != 300 {
		t.Fatalf("compaction=%v known=%v", got, ok)
	}
	// Snapshot pointers are detached from both the next snapshot and the journal.
	*timing.StartedAt = 99999
	if got, _ := trajectoryFixtureFind(t, s, "assistant:1:1").DurationMillis(); got != 1000 {
		t.Fatalf("snapshot alias changed timing: %v", got)
	}
}

func TestTrajectoryMissingInvalidAndPagedBoundaries(t *testing.T) {
	s := New()
	s.applyPersistenceEvent(trajectoryFixtureAssistant(3, 2000, 1, 1, nil))
	if _, ok := trajectoryFixtureFind(t, s, "assistant:1:1").DurationMillis(); ok {
		t.Fatal("missing step/start fabricated duration")
	}
	raw := trajectoryFixtureEvent(1, "step/start", 1000, map[string]any{"turn": 1, "step": 1})
	var record api.SessionPageRecordsItem
	b, _ := json.Marshal(map[string]any{"type": "event", "event": json.RawMessage(raw)})
	if err := json.Unmarshal(b, &record); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.prependPageRecordsLocked([]api.SessionPageRecordsItem{record})
	s.mu.Unlock()
	if got, ok := trajectoryFixtureFind(t, s, "assistant:1:1").DurationMillis(); !ok || got != 1000 {
		t.Fatalf("page boundary not hydrated: %v %v", got, ok)
	}
	s.applyPersistenceEvent(trajectoryFixtureEvent(4, "step/start", 3000, map[string]any{"turn": 1, "step": 2}))
	s.applyPersistenceEvent(trajectoryFixtureAssistant(5, 2500, 1, 2, nil))
	if _, ok := trajectoryFixtureFind(t, s, "assistant:1:2").DurationMillis(); ok {
		t.Fatal("negative clock interval became duration")
	}
	s.applyPersistenceEvent(trajectoryFixtureEvent(6, "step/start", nil, map[string]any{"turn": 2, "step": 1}))
	s.applyPersistenceEvent(trajectoryFixtureAssistant(7, 4000, 2, 1, nil))
	if _, ok := trajectoryFixtureFind(t, s, "assistant:2:1").DurationMillis(); ok {
		t.Fatal("missing timestamp became zero epoch")
	}
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"assistant/message","seq":3,"time":2000,"data":{"turn":1,"step":1,"message":{"id":"new","content":[{"type":"text","text":"new"}]}}}}]}`))
	if _, ok := trajectoryFixtureFind(t, s, "assistant:1:1").DurationMillis(); ok {
		t.Fatal("replacement snapshot reused old boundary")
	}
}

func TestTrajectoryLiveFirstTokenAndIncompleteClosure(t *testing.T) {
	s := New()
	s.applyPersistenceEvent(trajectoryFixtureEvent(1, "step/start", 1000, map[string]any{"turn": 1, "step": 1}))
	s.mu.Lock()
	a := &assistantStreamAttempt{AttemptID: "a", Turn: 1, Step: 1}
	s.assistantStream.active = a
	s.applyAssistantChunkLocked(a, json.RawMessage(`{"type":"block-start","index":0,"blockType":"text"}`), 1010)
	s.applyAssistantChunkLocked(a, json.RawMessage(`{"type":"tool-call-delta","index":1,"id":"t","name":"fixture","argumentsDelta":""}`), 1100)
	s.applyAssistantChunkLocked(a, json.RawMessage(`{"type":"text-delta","index":0,"text":"answer"}`), 1200)
	s.mu.Unlock()
	timing := trajectoryFixtureFind(t, s, "assistant:1:1")
	if _, ok := timing.DurationMillis(); ok {
		t.Fatal("live operation fabricated completed duration")
	}
	if timing.FirstTokenAt == nil || *timing.FirstTokenAt != 1100 {
		t.Fatalf("name-bearing tool delta should own first token: %+v", timing)
	}
	s.mu.Lock()
	changed := s.applyPersistenceEventLocked(trajectoryFixtureEvent(2, "step/end", 1400, map[string]any{"turn": 1, "step": 1}))
	s.mu.Unlock()
	if !changed {
		t.Fatal("timing-only boundary failed to notify UI")
	}
	timing = trajectoryFixtureFind(t, s, "assistant:1:1")
	if !timing.Closed {
		t.Fatal("step/end did not close operation")
	}
	if _, ok := timing.DurationMillis(); ok {
		t.Fatal("step/end substituted for missing assistant completion")
	}
	s.mu.Lock()
	s.invalidateFollowLocked()
	s.mu.Unlock()
	s.applyPersistenceEvent(trajectoryFixtureAssistant(3, 2000, 1, 1, nil))
	if _, ok := trajectoryFixtureFind(t, s, "assistant:1:1").DurationMillis(); ok {
		t.Fatal("invalidated follow kept prior start")
	}
}
