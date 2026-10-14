package session

import (
	"encoding/json"
	"testing"
)

func TestApplyTurnTimingEvent(t *testing.T) {
	st := New()
	start, _ := json.Marshal(map[string]any{
		"type": "turn/start", "seq": 1, "time": 1_000_000.0, "data": map[string]any{"turn": 0},
	})
	end, _ := json.Marshal(map[string]any{
		"type": "turn/end", "seq": 2, "time": 1_078_000.0, "data": map[string]any{"turn": 0},
	})
	st.applyPersistenceEvent(start)
	st.applyPersistenceEvent(end)
	snap := st.Snapshot()
	if len(snap.Messages) != 1 {
		t.Fatalf("messages = %d", len(snap.Messages))
	}
	msg := snap.Messages[0]
	if msg.Role != "status" || msg.EventType != "turn/end" || msg.Body != "78000" {
		t.Fatalf("%#v", msg)
	}
}
