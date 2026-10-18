package chat

import "testing"

func TestQuoteIntoDraft(t *testing.T) {
	draft := "next"
	QuoteIntoDraft(&draft, "hello\nworld")
	want := "> hello\n> world\n\nnext"
	if draft != want {
		t.Fatalf("got %q want %q", draft, want)
	}
}

func TestQuoteIntoDraftEmpty(t *testing.T) {
	draft := "x"
	QuoteIntoDraft(&draft, "  ")
	if draft != "x" {
		t.Fatalf("%q", draft)
	}
}
