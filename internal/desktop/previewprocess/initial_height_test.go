package previewprocess

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Memory wire fixtures do not initialize a preview platform or start a helper.
func TestInitialViewportHeightProtocolAndLegacyDefault(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	for _, height := range []int{0, 80, 480, 912, 2000} {
		doc := Document{HTML: "<p>fixture</p>", Width: 728, InitialHeight: height, Appearance: "light", Inline: true}
		var wire bytes.Buffer
		if err := sendDocument(&wire, nonce, doc); err != nil {
			t.Fatal(err)
		}
		got, err := receiveDocument(&wire, nonce)
		if err != nil || got != doc || wire.Len() != 0 {
			t.Fatalf("height=%d doc=%+v error=%v", height, got, err)
		}
		want := height
		if height == 0 {
			want = 700
		}
		if initialViewportHeight(got) != want {
			t.Fatalf("height=%d initial viewport=%d", height, initialViewportHeight(got))
		}
	}
}

func TestInitialViewportHeightRejectedAtSenderAndReceiver(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	for _, height := range []int{-1, 1, 79, 2001} {
		doc := Document{HTML: "x", Width: 728, InitialHeight: height, Appearance: "dark"}
		var wire bytes.Buffer
		if err := sendDocument(&wire, nonce, doc); err == nil || wire.Len() != 0 {
			t.Fatalf("sender accepted height %d", height)
		}
		if err := writePacket(&wire, packet{Version: protocolVersion, Kind: "document", Nonce: nonce, Bytes: 1, Width: 728, Height: height, Appearance: "dark"}); err != nil {
			t.Fatal(err)
		}
		wire.WriteByte('x')
		if _, err := receiveDocument(&wire, nonce); err == nil {
			t.Fatalf("receiver accepted height %d", height)
		}
	}
}

// This source-boundary check supplements the wire fixtures. It establishes
// wiring only; it does not claim Windows/WebView2 rendering acceptance.
func TestWindowsHelperUsesInitialViewportBeforeDocumentLoad(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(here), "helper_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	viewport := strings.Index(source, "w.SetContentSize(doc.Width, initialViewportHeight(doc))")
	load := strings.Index(source, "encoded, err := json.Marshal(shell)")
	if viewport < 0 || load < viewport {
		t.Fatal("helper does not apply the authorized initial viewport before loading its shell")
	}
	if !strings.Contains(source, `Math.max(80,Math.min(2000,Math.ceil(window.__previewState?.height`) || !strings.Contains(source, `Kind: "size"`) {
		t.Fatal("helper no longer publishes bounded content measurements after initialization")
	}
}
