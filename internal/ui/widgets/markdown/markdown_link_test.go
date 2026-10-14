package markdown

import "testing"

func TestLinksExtractsHTTPS(t *testing.T) {
	src := "See [docs](https://example.com/a) and also https://ignored.example because bare URLs need autolink."
	links := Links(src)
	if len(links) != 1 || links[0].URL != "https://example.com/a" || links[0].Label != "docs" {
		t.Fatalf("%+v", links)
	}
}
