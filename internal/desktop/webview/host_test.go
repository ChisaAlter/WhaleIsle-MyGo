package webview

import "testing"

func TestValidateHTTPURL(t *testing.T) {
	if err := validateHTTPURL("http://127.0.0.1/"); err != nil {
		t.Fatal(err)
	}
	if err := validateHTTPURL("file:///tmp"); err == nil {
		t.Fatal("expected reject")
	}
}

func TestURLOrigin(t *testing.T) {
	if got := urlOrigin("http://127.0.0.1:3080/path"); got != "http://127.0.0.1:3080" {
		t.Fatalf("got %q", got)
	}
}
