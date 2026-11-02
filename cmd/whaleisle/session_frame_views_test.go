package main

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestSessionFrameViewsShareHandlesWithinPassOnly(t *testing.T) {
	first, second := session.New(), session.New()
	first.Select("primary")
	second.Select("replacement")
	views := make(sessionFrameViews)
	defer views.end()
	if views.view(nil) != nil || len(views) != 0 {
		t.Fatal("missing session Store must not register an invalid frame handle")
	}
	primary := views.view(first)
	if primary != views.view(first) || !primary.BelongsTo(first) {
		t.Fatal("same pass did not return the existing primary handle")
	}
	if got := views.snapshot(first).SelectedID; got != "primary" {
		t.Fatal("primary snapshot mismatch")
	}
	replacement := views.view(second)
	if replacement == primary || !replacement.BelongsTo(second) || replacement.BelongsTo(first) {
		t.Fatal("Store replacement borrowed primary pass")
	}
	if got := views.snapshot(second).SelectedID; got != "replacement" {
		t.Fatal("replacement snapshot mismatch")
	}
	first.Select("primary-next")
	if got := views.snapshot(first).SelectedID; got != "primary-next" {
		t.Fatal("same pass ignored selection revision")
	}
	otherPass := make(sessionFrameViews)
	defer otherPass.end()
	if otherPass.view(first) == primary {
		t.Fatal("independent render passes shared a handle")
	}
	otherPass.snapshot(first)
	views.end()
	if got := otherPass.snapshot(first).SelectedID; got != "primary-next" {
		t.Fatal("ending one pass invalidated another")
	}
}
