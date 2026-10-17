package session

import (
	"testing"
	"time"
)

func TestUpdateQueueNoClientSetsLastAction(t *testing.T) {
	s := New()
	s.UpdateQueue(t.Context(), "sid", "item-1", QueueActionRemove)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snap := s.Snapshot(); snap.LastAction != "" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected lastAction after UpdateQueue without client")
}
