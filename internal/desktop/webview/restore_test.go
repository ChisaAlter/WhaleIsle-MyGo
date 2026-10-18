package webview

import "testing"

func TestRestorePanelURL(t *testing.T) {
	const old = "http://127.0.0.1:3080"
	const current = "http://127.0.0.1:4080"
	for _, tt := range []struct{ name, origin, previousOrigin, saved, want string }{
		{"first open", current, "", "", current + "/"},
		{"restart retains panel", current, old, old + "/plugins/view?desktopPlugin=dshbot&session=abc#details", current + "/plugins/view?desktopPlugin=dshbot&session=abc#details"},
		{"external navigation", current, old, "https://example.com/private", current + "/"},
		{"credentials rejected", current, old, "http://user:secret@127.0.0.1:3080/", current + "/"},
		{"runtime unavailable", "", old, old + "/", ""},
		{"unsafe runtime", "file:///tmp/index", old, old + "/", ""},
		{"unrecorded origin", current, "", old + "/?desktopPlugin=dshbot", current + "/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := RestorePanelURL(tt.origin, tt.previousOrigin, tt.saved); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
