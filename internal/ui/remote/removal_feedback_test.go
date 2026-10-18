package remote

import (
	"testing"
	remotestore "whaleisle/internal/store/remote"
)

func TestUpstreamDeltaRemovalConfirmation(t *testing.T) {
	e := MachineEditor{Open: true, Draft: remotestore.Draft{ID: "one", Name: "unsaved draft"}, PendingRemove: "one", RemovalGeneration: 3, Removing: true}
	e.syncRemoval(remotestore.Snapshot{Generation: 3, Busy: true})
	if e.PendingRemove != "one" || !e.Removing {
		t.Fatal("pending request dismissed")
	}
	e.syncRemoval(remotestore.Snapshot{Generation: 3, Error: "fixture failure"})
	if e.PendingRemove != "one" || e.Removing || !e.Open || e.Draft.Name != "unsaved draft" {
		t.Fatalf("failure lost confirmation/draft: %+v", e)
	}
	e.PendingRemove = "" // Cancel confirmation returns to unchanged draft.
	e.syncRemoval(remotestore.Snapshot{Generation: 3})
	if !e.Open || e.Draft.Name != "unsaved draft" {
		t.Fatal("cancel lost draft")
	}
	e.PendingRemove = "one"
	e.Removing = true
	e.syncRemoval(remotestore.Snapshot{Generation: 3, RemovedID: "one", MutationApplied: true})
	if e.PendingRemove != "" || e.Open || e.Draft.ID != "" {
		t.Fatalf("successful delete retained stale editor: %+v", e)
	}
	e = MachineEditor{Open: true, Draft: remotestore.Draft{ID: "one", Name: "new draft"}, PendingRemove: "one", RemovalGeneration: 3, Removing: true}
	e.syncRemoval(remotestore.Snapshot{Generation: 4, RemovedID: "one", MutationApplied: true})
	if e.PendingRemove != "" || e.Removing || !e.Open || e.Draft.Name != "new draft" {
		t.Fatal("old-host removal cleared current draft")
	}
}
