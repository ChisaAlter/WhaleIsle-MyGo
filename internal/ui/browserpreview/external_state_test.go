package browserpreview

import (
	"errors"
	"testing"
)

func TestOpenBrowserAddressUsesExplicitDraft(t *testing.T) {
	var opened []string
	open := func(raw string) error { opened = append(opened, raw); return nil }
	for _, bad := range []string{"", "  ", "file:///tmp/test.html", "javascript:alert(1)", "data:text/html,test", "ftp://example.test/", "https://user:password@example.test/", "https://", "https://example.test/\nnext"} {
		if err := openBrowserAddress(bad, open); err == nil {
			t.Fatalf("must reject invalid/credential-bearing address %q", bad)
		}
	}
	if len(opened) != 0 {
		t.Fatal("invalid input must never reach the external opener")
	}
	if err := openBrowserAddress(" example.test/typed?x=1#part ", open); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://example.test/typed?x=1#part" {
		t.Fatalf("must open exactly the current normalized address draft once: %v", opened)
	}
	if err := openBrowserAddress("http://localhost:3000/", open); err != nil || len(opened) != 2 || opened[1] != "http://localhost:3000/" {
		t.Fatal("explicit HTTP development addresses must remain supported")
	}
	want := errors.New("synthetic system opener failure")
	calls := 0
	err := openBrowserAddress("https://example.test/", func(string) error { calls++; return want })
	if !errors.Is(err, want) || calls != 1 {
		t.Fatal("surface the system error without retrying")
	}
	if openBrowserAddress("https://example.test/", nil) == nil {
		t.Fatal("an unavailable system opener must fail closed")
	}
}
