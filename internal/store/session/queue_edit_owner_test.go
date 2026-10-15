package session

import (
	"context"
	"testing"
	"whaleisle/internal/dsh"
)

func TestQueueEditRejectsOldRenderedSession(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "B"
	s.BeginQueueEditForSession(context.Background(), "A", "shared-item")
	if got := s.Snapshot().QueueEditor; got.ItemID != "" || got.Loading || got.Ready {
		t.Fatalf("stale frame loaded another session: %+v", got)
	}
}
