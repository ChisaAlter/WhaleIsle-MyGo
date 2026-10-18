package browserpreview

import "testing"

func TestBrowserClearConfirmation(t *testing.T) {
	var confirm *browserDataConfirmation[int]
	if takeBrowserDataConfirmation(&confirm, "cookies", 1, 7) {
		t.Fatal("cleanup must not run without explicit confirmation")
	}
	confirm = &browserDataConfirmation[int]{kind: "cookies", host: 1, epoch: 7}
	for _, mismatch := range []struct {
		kind  string
		host  int
		epoch uint64
	}{{"cache", 1, 7}, {"cookies", 2, 7}, {"cookies", 1, 8}, {"all", 1, 7}} {
		if takeBrowserDataConfirmation(&confirm, mismatch.kind, mismatch.host, mismatch.epoch) || confirm == nil {
			t.Fatal("wrong cleanup, Host, or navigation must not consume the confirmation")
		}
	}
	calls := 0 // Synthetic dispatch only. No native browser or cookies exist.
	for range 2 {
		if takeBrowserDataConfirmation(&confirm, "cookies", 1, 7) {
			calls++
		}
	}
	if calls != 1 || confirm != nil {
		t.Fatal("one explicit confirmation must permit exactly one cleanup")
	}
	confirm = &browserDataConfirmation[int]{kind: "cache", host: 1, epoch: 9}
	if !takeBrowserDataConfirmation(&confirm, "cache", 1, 9) {
		t.Fatal("the existing HTTP-cache confirmation must retain its own scope")
	}
}
