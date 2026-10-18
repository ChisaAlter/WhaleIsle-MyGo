package session

import (
	"context"
	"testing"
)

func TestInlinePublicationReadDoesNotReplaceSiblingOrSource(t *testing.T) {
	store, source, _ := interactiveFixtureStore(t)
	store.visualReply.VisualReplyState = VisualReplyState{SessionID: "A", AttachmentID: "other", HTML: "existing source", Ready: true}
	store.visualReply.generation = store.followGeneration
	html, valid, err := store.ReadVisualReplySnapshot(context.Background(), "A", "page")
	if err != nil || html != source || !valid() {
		t.Fatalf("independent read: %v", err)
	}
	if got := store.VisualReplySnapshot(); got.AttachmentID != "other" || got.HTML != "existing source" {
		t.Fatal("independent read replaced another page")
	}
	store.mu.Lock()
	store.followGeneration++
	store.mu.Unlock()
	if valid() {
		t.Fatal("independent context survived Host generation change")
	}
	if _, _, err := store.ReadVisualReplySnapshot(context.Background(), "A", "unknown"); err == nil {
		t.Fatal("accepted unpublished attachment")
	}
}
