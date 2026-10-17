package session

import (
	"encoding/json"
	"testing"
)

func TestParseBandActions(t *testing.T) {
	tree := []json.RawMessage{
		json.RawMessage(`{"actionId":"accept","label":"Accept"}`),
		json.RawMessage(`{"children":[{"id":"reject","title":"Reject"}]}`),
	}
	got := parseBandActions(tree)
	if len(got) != 2 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if got[0].ID != "accept" || got[0].Label != "Accept" {
		t.Fatalf("first %#v", got[0])
	}
	if got[1].ID != "reject" || got[1].Label != "Reject" {
		t.Fatalf("second %#v", got[1])
	}
}
