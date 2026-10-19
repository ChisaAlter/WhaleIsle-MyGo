package main

import (
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
