package session

import (
	"encoding/json"
	"testing"
)

func TestQueueAttachmentPreview(t *testing.T) {
	s := New()
	s.selectedID = "s"
	s.applyQueueInboxLocked(json.RawMessage(`{"next-turn":[{"id":"q","content":[{"type":"text","text":"read"},{"type":"file","attachment":{"attachmentId":"f","name":"report.pdf","bytes":2048,"mediaType":"application/pdf"}}]}]}`), 1)
	if s.planBoard.Queue[0].Preview != "read" {
		t.Fatalf("attachment should have its own identity, preview=%q", s.planBoard.Queue[0].Preview)
	}
}
func TestQueueActionGatesPreserveLegalOperations(t *testing.T) {
	a := QueueActionState{Mutable: true}
	image := PlanQueueItem{ID: "q", ContentKnown: true, TextOnly: false, Attachments: []QueueAttachment{{Kind: "image", AttachmentID: "i"}}}
	if a.Can(QueueActionEdit, image) || a.Can(QueueActionSteer, image) || !a.Can(QueueActionRemove, image) {
		t.Fatal("idle attachment gates")
	}
	a.Running = true
	if !a.Can(QueueActionSteer, image) {
		t.Fatal("running attachment steer is valid")
	}
	text := PlanQueueItem{ID: "t", ContentKnown: true, TextOnly: true}
	if !a.Can(QueueActionEdit, text) || !a.Can(QueueActionEdit, PlanQueueItem{ID: "legacy"}) {
		t.Fatal("valid or metadata-pending text edit blocked")
	}
	a.Busy = true
	for _, action := range []UpdateQueueAction{QueueActionEdit, QueueActionRemove, QueueActionSteer} {
		if a.Can(action, text) {
			t.Fatalf("busy %s", action)
		}
	}
}
func TestQueueAttachmentIdentityAndImmutableSnapshot(t *testing.T) {
	s := New()
	s.selectedID = "s"
	s.applyQueueInboxLocked(json.RawMessage(`{"next-turn":[{"id":"q","source":{"kind":"user","rpcId":"r"},"content":[{"type":"file","attachment":{"attachmentId":"f","name":"report.pdf","bytes":2048}},{"type":"image","attachment":{"attachmentId":"i","width":20,"height":10,"bytes":42}},{"type":"image"}]}]}`), 1)
	q := s.Snapshot().PlanBoard.Queue[0]
	if q.TextOnly || !q.ContentKnown || q.RequestID != "r" || len(q.Attachments) != 2 || q.Attachments[0].Name != "report.pdf" || q.Attachments[1].Width != 20 {
		t.Fatalf("%+v", q)
	}
	q.Attachments[0].Name = "changed"
	if s.Snapshot().PlanBoard.Queue[0].Attachments[0].Name != "report.pdf" {
		t.Fatal("snapshot aliased queue metadata")
	}
}
