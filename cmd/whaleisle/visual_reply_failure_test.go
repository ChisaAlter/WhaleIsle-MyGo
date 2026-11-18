package main

import (
	"sync/atomic"
	"testing"

	"whaleisle/internal/store/session"
)

// Seed an already failed inline occurrence: no read, native window, preview
// process, or HTML execution is needed to exercise the actual Retry wiring.
func TestInlineVisualReplyOffersOnlyRetryableFailureAction(t *testing.T) {
	store := session.New()
	handler := inlinePublishedHTMLHandler(nil, store, "fixture", false)
	if handler == nil {
		t.Skip("inline surfaces unavailable on this target")
	}
	for _, retryable := range []bool{false, true} {
		key := inlinePageKey{store: store, sessionID: "fixture", attachmentID: "page"}
		page := &inlinePage{state: session.InlineVisualReply{Error: "fixture failure"}, retryable: retryable}
		inlinePages.Lock()
		inlinePages.pages[key] = page
		inlinePages.Unlock()
		state := handler("page")
		inlinePages.Lock()
		delete(inlinePages.pages, key)
		inlinePages.Unlock()
		if (state.Retry != nil) != retryable {
			t.Fatalf("Retry present=%v, retryable=%v", state.Retry != nil, retryable)
		}
	}
}

func TestRetireStaleInlineVisualPagesEvictsInvalidWithoutTouchingCurrent(t *testing.T) {
	store := session.New()
	staleKey := inlinePageKey{store: store, sessionID: "old-session", attachmentID: "stale"}
	activeKey := inlinePageKey{store: store, sessionID: "cur-session", attachmentID: "active"}
	var staleRetired, activeRetired atomic.Int32

	inlinePages.Lock()
	inlinePages.pages[staleKey] = &inlinePage{
		valid:  func() bool { return false },
		retire: func() { staleRetired.Add(1) },
	}
	inlinePages.pages[activeKey] = &inlinePage{
		valid:  func() bool { return true },
		retire: func() { activeRetired.Add(1) },
	}
	inlinePages.Unlock()
	t.Cleanup(func() {
		inlinePages.Lock()
		delete(inlinePages.pages, staleKey)
		delete(inlinePages.pages, activeKey)
		inlinePages.Unlock()
	})

	(&mainState{}).selectMainPanel("assistant")

	if staleRetired.Load() != 1 {
		t.Fatalf("stale retire callbacks = %d, want 1", staleRetired.Load())
	}
	if activeRetired.Load() != 0 {
		t.Fatalf("active retire callbacks = %d, want 0", activeRetired.Load())
	}
	inlinePages.Lock()
	_, stalePresent := inlinePages.pages[staleKey]
	_, activePresent := inlinePages.pages[activeKey]
	inlinePages.Unlock()
	if stalePresent {
		t.Fatal("stale inline page remained registered after panel switch")
	}
	if !activePresent {
		t.Fatal("active inline page was evicted unexpectedly")
	}
}
