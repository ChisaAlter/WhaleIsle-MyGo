package sidebar

import (
	"testing"

	"whaleisle/internal/store/session"
)

func TestFilterItemsByTitle(t *testing.T) {
	items := []session.Item{
		{SessionID: "a", Title: "Alpha"},
		{SessionID: "b", Title: "Beta", ParentID: "a"},
		{SessionID: "c", Title: "Gamma"},
	}
	got := filterItems(items, "beta")
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (parent + child)", len(got))
	}
}

func TestSidebarItemsUsesRPCHits(t *testing.T) {
	snap := session.ViewState{
		Items: []session.Item{
			{SessionID: "a", Title: "Alpha"},
			{SessionID: "b", Title: "Beta", ParentID: "a"},
			{SessionID: "c", Title: "Gamma"},
		},
		SearchRPCQuery: "q",
		SearchHitIDs:   map[string]bool{"b": true},
	}
	q := "q"
	got := sidebarItems(snap, &q)
	if len(got) != 2 {
		t.Fatalf("len = %d, want parent+child for hit b", len(got))
	}
}

func TestFilterItemsEmptyQuery(t *testing.T) {
	items := []session.Item{{SessionID: "a", Title: "A"}}
	got := filterItems(items, "  ")
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
}
