package previewprocess

import (
	"bytes"
	"strings"
	"testing"
)

func TestFailureReceiptReportsOnlyAuthenticatedFixedPhaseAndCode(t *testing.T) {
	nonce := strings.Repeat("b", 64)
	p := packet{Version: protocolVersion, Kind: "failure", Nonce: nonce, FailureStage: "document-ready", FailureCode: "deadline-exceeded"}
	var wire bytes.Buffer
	if err := writePacket(&wire, p); err != nil {
		t.Fatal(err)
	}
	decoded, err := readPacket(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if err := expect(decoded, "loaded", nonce); err == nil || err.Error() != "preview helper document-ready: deadline-exceeded" {
		t.Fatalf("lost fixed document-load failure: %v", err)
	}
	for _, mutate := range []func(*packet){
		func(p *packet) { p.Nonce = strings.Repeat("c", 64) },
		func(p *packet) { p.FailureStage = "untrusted page content" },
		func(p *packet) { p.FailureCode = "secret=never retain" },
		func(p *packet) { p.Kind = "loaded" },
	} {
		bad := p
		mutate(&bad)
		if err := expect(bad, "loaded", nonce); err == nil || strings.Contains(err.Error(), "untrusted") || strings.Contains(err.Error(), "secret=") {
			t.Fatalf("invalid receipt accepted or reflected document text: %v", err)
		}
	}
}
