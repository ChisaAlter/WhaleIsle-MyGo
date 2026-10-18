package dshbot

import (
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestBotRoutineAdvancedCanonicalAndUnavailable(t *testing.T) {
	d := RoutineAdvancedDraft{ContextFrom: []string{"missing"}, WatchUnknown: true, WatchRaw: `{"kind":"future"}`}
	patch, err := prepareRoutineAdvanced(d, "b", "r", []pluginrpc.DshbotRoutine{{ID: "other", BotID: "b"}})
	if err == nil || patch.ContextSet || patch.WatchSet || d.ContextFrom[0] != "missing" {
		t.Fatal("save would normalize away untouched unresolved context")
	}
	c, key := advancedReady(t, true)
	c.Edit(key, "watchValue", "  fixture-only-command-normalized  ")
	c.Edit(key, "watchTimeout", "015000")
	c.Prepare(key, "save")
	snapshot := c.Snapshot(key)
	if snapshot.Review == nil {
		t.Fatal(snapshot.Error)
	}
	if snapshot.Review.Advanced.WatchValue != "fixture-only-command-normalized" || snapshot.Review.Advanced.WatchTimeout != "15000" || snapshot.Review.Command.Advanced.Watch.Value != snapshot.Review.Advanced.WatchValue {
		t.Fatal("review differs from canonical payload")
	}
	if snapshot.Draft.Advanced.WatchValue != "  fixture-only-command-normalized  " || snapshot.Draft.Advanced.WatchTimeout != "015000" {
		t.Fatal("raw draft destroyed")
	}
	if !RoutineReviewNeedsWatchAcknowledgement(*snapshot.Review) {
		t.Fatal("normalization bypassed command acknowledgement")
	}
}
