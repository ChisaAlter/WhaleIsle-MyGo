package markdown

import "testing"

func TestLinksExtractsHTTPS(t *testing.T) {
	src := "See [docs](https://example.com/a) and also https://plain.example with a bare URL."
	links := Links(src)
	if len(links) != 2 || links[0].URL != "https://example.com/a" || links[0].Label != "docs" {
		t.Fatalf("%+v", links)
	}
}
