package openui

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestGenuiMediaHasConstrainedNativeResourceConsumer(t *testing.T) {
	e := Event{InstanceID: Identity("host", "session", "message", 0), RequestID: "media", Path: "/api/media", Query: map[string]string{"src": "http://127.0.0.1/private"}}
	if err := validateAPIRequest(e); err != nil {
		t.Fatal("fixed media API rejected", err)
	}
	if got := (&ResourceAPI{}).Handle(context.Background(), e); got.Status != 403 {
		t.Fatal("private media source not blocked", got.Status)
	}
	wav := append([]byte("RIFF"), 0, 0, 0, 0)
	wav = append(wav, []byte("WAVEfmt ")...)
	wav = append(wav, make([]byte, 64)...)
	got := publicMediaResponse(e, publicResource{data: wav, source: "https://example.invalid/sample.wav"})
	if got.Status != 200 || !bytes.Contains(got.Body, []byte("data:audio/wave;base64,")) {
		t.Fatal("typed media body missing", got.Status, string(got.Body))
	}
	if got := publicMediaResponse(e, publicResource{data: []byte("<script>not media</script>")}); got.Status != 415 {
		t.Fatal("non-media body passed")
	}
	for _, fragment := range []string{"/api/media", "audio[src],video[src]", "prepareNestedResources"} {
		if !strings.Contains(string(whaleNativeAdapter), fragment) {
			t.Fatal("Genui remote resource consumer absent", fragment)
		}
	}
}
func TestNativeWhaleHtmlViewReusesShippedThemeAndModuleComponent(t *testing.T) {
	for _, fragment := range []string{"const upstreamHtmlView=nCt.components.HtmlView.component", "component:NativeShippedHtmlView", "Waiting for complete HTML content"} {
		if !strings.Contains(whaleNativeAdapter, fragment) {
			t.Fatal("shipped HtmlView consumer absent", fragment)
		}
	}
	for _, fragment := range []string{"Math.max(40,Math.min(12e3", "dataset.theme", "--whale-fg", "sandbox:\"allow-scripts\""} {
		if !strings.Contains(whaleViewer, fragment) {
			t.Fatal("baseline HtmlView behavior changed", fragment)
		}
	}
	shell, err := whaleShell(whaleViewer, Render{InstanceID: Identity("h", "s", "m", 0), Program: "Text(\"fixture\")"}, 800, "dark")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"https://cdn.jsdelivr.net", "https://unpkg.com", "https://esm.sh", "https://cdnjs.cloudflare.com"} {
		if !strings.Contains(shell, source) {
			t.Fatal("original script CDN blocked", source)
		}
	}
	if !strings.Contains(shell, "connect-src &#39;none&#39;") {
		t.Fatal("arbitrary network access enabled")
	}
}
