package htmlpreview

import (
	"html"
	"strings"
	"testing"
)

func TestInertImageInheritedPolicyAllowsOnlyInlineMedia(t *testing.T) {
	image, err := ProcessImageDocumentShell(`<img src="data:image/svg+xml;base64,PHN2Zy8+">`, 300, "light")
	if err != nil {
		t.Fatal(err)
	}
	end := strings.Index(image, "><style>")
	if end < 0 {
		t.Fatal("outer policy not found")
	}
	policy := html.UnescapeString(image[:end])
	for _, required := range []string{"default-src 'none'", "img-src data:", "font-src data:", "connect-src 'none'", "worker-src 'none'", "object-src 'none'", "base-uri 'none'", "form-action 'none'"} {
		if !strings.Contains(policy, required) {
			t.Fatalf("inherited policy lost %q: %s", required, policy)
		}
	}
	for _, forbidden := range []string{"https:", "http:", "file:", "blob:", "allow-same-origin"} {
		if strings.Contains(policy, forbidden) {
			t.Fatalf("inherited image policy admitted %q", forbidden)
		}
	}
	ordinary, err := ProcessDocumentShell(`<p>ordinary</p>`, 300, "light")
	if err != nil {
		t.Fatal(err)
	}
	ordinaryEnd := strings.Index(ordinary, "><style>")
	if ordinaryEnd < 0 || strings.Contains(html.UnescapeString(ordinary[:ordinaryEnd]), "img-src data:") {
		t.Fatal("inert-only media policy escaped into ordinary document mode")
	}
}
