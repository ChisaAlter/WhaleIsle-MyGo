package files

import "testing"

func TestTextReadLineDefersUntilOwnedBodyReady(t *testing.T) {
	var first, other textReadLine
	first.begin()
	if !first.deferRequest(180) || first.line != 180 {
		t.Fatal("raw line was clamped before text arrived")
	}
	if got := first.complete(false); got != 0 || first.line != 180 || !first.waiting {
		t.Fatal("failure consumed the request instead of retaining it for retry")
	}
	// Switching tabs must not publish the first tab's location to the other.
	other.begin()
	if got := other.complete(true); got != 0 || first.line != 180 {
		t.Fatal("pending line crossed tab ownership")
	}
	// Cancel has no successful completion. A later explicit retry keeps the line.
	first.begin()
	if got := first.complete(true); got != 180 {
		t.Fatalf("successful body lost original target: %d", got)
	}
	if got := first.complete(true); got != 0 || first.deferRequest(9) {
		t.Fatal("completed request repeated or ready text still deferred")
	}
	first.begin()
	first.deferRequest(20)
	first.deferRequest(70)
	if got := first.complete(true); got != 70 {
		t.Fatal("latest explicit line request did not win")
	}
	first.begin()
	first.deferRequest(99)
	first.reset() // Host binding changed, or the old tab was discarded.
	if got := first.complete(true); got != 0 {
		t.Fatal("old Host line survived owner reset")
	}
	var reopened textReadLine
	reopened.begin()
	if got := reopened.complete(true); got != 0 {
		t.Fatal("same-path reopened tab inherited a closed tab's request")
	}
}
