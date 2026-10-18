package mygo

import "testing"

func TestDisabledBindingsRejectEveryOriginAndBroadcast(t *testing.T) {
	w := &Window{disableBindings: true, trustedOrigins: []string{"*"}}
	for _, origin := range []string{"about:blank", "file:///tmp/test.html", "https://example.com", "http://localhost:3000", "mygo://app"} {
		if w.isTrusted(origin) {
			t.Fatalf("isolated page trusted %s", origin)
		}
	}
	w.enqueue(message{head: []byte("secret")}, true)
	if len(w.held) != 0 || len(w.outbox) != 0 {
		t.Fatal("isolated window received app data")
	}
	normal := &Window{trustedOrigins: []string{"https://trusted.example"}}
	if !normal.isTrusted("https://trusted.example") || !normal.isTrusted("about:blank") {
		t.Fatal("normal trust behavior changed")
	}
}
