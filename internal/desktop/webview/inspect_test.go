package webview

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestPickedAnnotationTreatsPageTextAsQuotedData(t *testing.T) {
	p := PickedElement{URL: "https://example.com/", Tag: "button", Selector: "#send", Text: "hello\n\000world"}
	got := p.Markdown()
	if !strings.Contains(got, "> Text: hello world") || strings.Contains(got, "\033") {
		t.Fatalf("unbounded or unquoted annotation: %q", got)
	}
	if !strings.Contains(got, "Browser element selection") || !strings.Contains(got, "> Selector: #send") {
		t.Fatalf("missing selection context: %q", got)
	}
}

func TestValidatePickedElementRejectsInvalidBoundsAndPayload(t *testing.T) {
	p := PickedElement{Tag: "button", Selector: "#send", Width: 12, Height: 20}
	if err := validatePickedElement(p); err != nil {
		t.Fatal(err)
	}
	p.Width = -1
	if validatePickedElement(p) == nil {
		t.Fatal("negative rectangle accepted")
	}
	p.Width = 12
	p.Text = strings.Repeat("x", 2049)
	if validatePickedElement(p) == nil {
		t.Fatal("oversized page text accepted")
	}
}

func TestValidateBrowserCaptureRejectsNonPNGAndOversize(t *testing.T) {
	if validateBrowserCapture([]byte("not a png")) == nil {
		t.Fatal("non-PNG accepted")
	}
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	if err := validateBrowserCapture(out.Bytes()); err != nil {
		t.Fatal(err)
	}
	if validateBrowserCapture(make([]byte, MaxBrowserScreenshotBytes+1)) == nil {
		t.Fatal("oversized bytes accepted")
	}
}
