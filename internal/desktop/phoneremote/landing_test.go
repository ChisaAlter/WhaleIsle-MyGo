package phoneremote

import "testing"

func TestLandingURLBoundary(t *testing.T) {
	for _, tc := range []struct{ name, base, want string }{
		{"https landing", "https://app.example.test/dshd/", "https://app.example.test/dshd"},
		{"lan landing", "http://192.0.2.10:3180/", "http://192.0.2.10:3180"},
		{"no daemon fallback", "", ""},
		{"script", "javascript:alert(1)", ""},
		{"file", "file:///tmp/mobile.html", ""},
		{"relative", "/dshd", ""},
		{"missing host", "https:///dshd", ""},
		{"credentials", "https://user:secret@app.example.test/", ""},
		{"offer fragment", "https://app.example.test/#offer=fixture", ""},
		{"query credentials", "https://app.example.test/?token=fixture", ""},
		{"wildcard", "http://0.0.0.0:3180", ""},
		{"bad port", "https://app.example.test:99999", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ConnectURL(Snapshot{RemoteAppBaseURL: tc.base, RemoteMode: "lan", RemoteBindAddress: "192.0.2.10", RemotePort: 8411})
			if got != tc.want {
				t.Fatalf("landing URL did not meet boundary for %s", tc.name)
			}
		})
	}
}
