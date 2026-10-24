package markdown

import "testing"

// Recorded from the current Node URL implementation used by the installed
// safeHref/linkLabel helpers; retains the raw destination independently of label.
func TestToolWebURLBrowserGoldens(t *testing.T) {
	for _, tc := range []struct {
		raw, label string
		safe       bool
	}{
		{"https://www.Example.test:443/path", "www.example.test", true},
		{" https://EXAMPLE.test/a ", "example.test", true},
		{"https:example.test/a", "example.test", true},
		{"http:\\example.test\\a", "example.test", true},
		{"http://0x7f.1/a", "127.0.0.1", true},
		{"http://0177.0.0.1/", "127.0.0.1", true},
		{"https://例子.测试/a", "xn--fsqu00a.xn--0zwm56d", true},
		{"https://[2001:0db8::1]:443/a", "[2001:db8::1]", true},
		{"https://user:pw@example.test/a", "example.test", true},
		{"https://example.test:99999/a", "https://example.test:99999/a", false},
		{"https://exa mple.test/", "https://exa mple.test/", false},
		{"//example.test/a", "//example.test/a", false},
		{"relative/path", "relative/path", false},
		{"ftp://example.test/a", "example.test", false},
		{"file:///tmp/a", "file:///tmp/a", false},
		{"data:text/plain,a", "data:text/plain,a", false},
		{"javascript:alert(1)", "javascript:alert(1)", false},
		{"mailto:a@example.test", "mailto:a@example.test", false},
		{"https://", "https://", false},
		{"\x00\tHTTPS://exa\nmple.test/a\r", "example.test", true},
		{"\ufeffhttps://example.test/a", "\ufeffhttps://example.test/a", false},
		{"https://%65xample.test/a", "example.test", true},
		{"http://1/", "0.0.0.1", true},
		{"http://256.1.1.1/", "http://256.1.1.1/", false},
		{"https://example.test./", "example.test.", true},
		{"https://example.test/?a=1#b", "example.test", true},
		{"", "", false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			want := ""
			if tc.safe {
				want = tc.raw
			}
			if got := WebHref(tc.raw); got != want {
				t.Fatalf("href%q expected%q", got, want)
			}
			if got := WebLinkLabel(tc.raw, ""); got != tc.label {
				t.Fatalf("label%q expected%q", got, tc.label)
			}
			for _, title := range []string{" ", "\t", "Title \n preserved", "\u00a0"} {
				if WebLinkLabel(tc.raw, title) != title {
					t.Fatal("title changed")
				}
			}
		})
	}
}
func TestToolWebCSSNormalWhitespace(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{"  a\t\r\nb\fc  ", "a b c"}, {"a\u00a0\u00a0b", "a\u00a0\u00a0b"}, {"\u0085a\u0085", "\u0085a\u0085"}, {"\t\r\n", ""}, {"a\vb", "a\vb"}} {
		if got := webInlineText(tc.raw); got != tc.want {
			t.Fatalf("%q =>%q expected%q", tc.raw, got, tc.want)
		}
	}
}
