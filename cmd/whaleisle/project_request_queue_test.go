package main

import (
	"testing"
	projectctl "whaleisle/internal/app/project"
	"whaleisle/internal/projectflow"
)

func TestProjectQueueUsesDisplayedRevisionAndBeforeIdentity(t *testing.T) {
	queue := &projectflow.RequestQueue{Queued: []projectflow.RequestQueueEntry{{ID: "a", Revision: 1}, {ID: "b", Revision: 2}, {ID: "c", Revision: 3}, {ID: "d", Revision: 4}}}
	page := projectflow.RequestPage{Items: []projectflow.RequestCard{{ID: "b", Revision: 12}}}
	for _, tc := range []struct {
		id, action, before string
		revision           int
	}{{"b", "moveUp", "a", 12}, {"b", "moveDown", "d", 12}, {"c", "moveDown", "", 3}, {"a", "cancel", "", 1}, {"b", "enqueue", "", 12}} {
		change, err := projectctl.QueueChange(page, nil, queue, tc.id, tc.action)
		if err != nil || change.RequestID != tc.id || change.RequestRevision != tc.revision || change.BeforeRequestID != tc.before {
			t.Fatalf("%s %s: %+v %v", tc.id, tc.action, change, err)
		}
		if (tc.action == "moveUp" || tc.action == "moveDown") && change.Action != "move" {
			t.Fatal("UI move direction leaked into Host protocol")
		}
	}
	detail := &projectflow.RequestCard{ID: "b", Revision: 20}
	change, err := projectctl.QueueChange(page, detail, queue, "b", "resume")
	if err != nil || change.RequestRevision != 20 {
		t.Fatal("fresh selected detail lost its revision to the list")
	}
	for _, tc := range []struct{ id, action string }{{"a", "moveUp"}, {"d", "moveDown"}, {"missing", "moveUp"}, {"b", "invented"}} {
		if _, err := projectctl.QueueChange(page, nil, queue, tc.id, tc.action); err == nil {
			t.Fatalf("invalid ordering accepted: %+v", tc)
		}
	}
	if _, err := projectctl.QueueChange(page, nil, nil, "b", "enqueue"); err == nil {
		t.Fatal("missing Host queue projection accepted a write")
	}
	if queue.Queued[1].ID != "b" || queue.Queued[1].Revision != 2 {
		t.Fatal("building a command mutated retained queue state")
	}
}
