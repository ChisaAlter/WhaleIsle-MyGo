package sidebar

import (
	"testing"

	"whaleisle/internal/store/session"
)

func TestBuildTreePinsFirst(t *testing.T) {
	items := []session.Item{
		{SessionID: "new", Title: "New", UpdatedAt: 10},
		{SessionID: "pin", Title: "Pinned", UpdatedAt: 1},
		{SessionID: "mid", Title: "Mid", UpdatedAt: 5},
	}
	nodes := buildTree(items, []string{"pin"})
	if len(nodes) != 3 {
		t.Fatalf("len = %d", len(nodes))
	}
	if nodes[0].item.SessionID != "pin" {
		t.Fatalf("first = %q", nodes[0].item.SessionID)
	}
}
