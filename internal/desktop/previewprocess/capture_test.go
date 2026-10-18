package previewprocess

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	"whaleisle/internal/desktop/htmlpreview"
)

func TestCaptureResultIPC(t *testing.T) {
	nonce := strings.Repeat("c", 64)
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewNRGBA(image.Rect(0, 0, 240, 1))); err != nil {
		t.Fatal(err)
	}
	result := htmlpreview.CaptureResult{PNG: base64.StdEncoding.EncodeToString(pngBytes.Bytes()), Width: 240, ContentHeight: 1, CapturedHeight: 1, ConsoleMessages: []htmlpreview.ConsoleMessage{{Level: "log", Text: "fixture"}}}
	var wire bytes.Buffer
	if err := sendCapture(&wire, nonce, result); err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), wire.Bytes()...)
	got, err := receiveCapture(&wire, nonce, 240)
	if err != nil || got.PNG != result.PNG || got.ConsoleMessages[0].Text != "fixture" || wire.Len() != 0 {
		t.Fatalf("capture roundtrip: %#v, %v", got, err)
	}
	for _, tc := range []struct {
		name  string
		data  []byte
		nonce string
		width int
	}{
		{"wrong nonce", raw, strings.Repeat("b", 64), 240},
		{"wrong width", raw, nonce, 241},
		{"truncated", raw[:len(raw)-1], nonce, 240},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := receiveCapture(bytes.NewReader(tc.data), tc.nonce, tc.width); err == nil {
				t.Fatal("accepted invalid capture")
			}
		})
	}
	wire.Reset()
	if err := writePacket(&wire, packet{Version: protocolVersion, Kind: "capture", Nonce: nonce, Bytes: maxCaptureBytes + 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := receiveCapture(&wire, nonce, 240); err == nil {
		t.Fatal("accepted oversized result")
	}
	result.PNG = base64.StdEncoding.EncodeToString([]byte("not a PNG"))
	wire.Reset()
	if err := sendCapture(&wire, nonce, result); err != nil {
		t.Fatal(err)
	}
	if _, err := receiveCapture(&wire, nonce, 240); err == nil {
		t.Fatal("accepted invalid PNG")
	}
}

func TestCaptureModeAndOptIn(t *testing.T) {
	nonce := strings.Repeat("c", 64)
	doc := Document{HTML: "<p>fixture</p>", Width: 240, Appearance: "dark", capture: true}
	var wire bytes.Buffer
	if err := sendDocument(&wire, nonce, doc); err != nil {
		t.Fatal(err)
	}
	got, err := receiveDocument(&wire, nonce)
	if err != nil || !got.capture {
		t.Fatalf("capture mode lost: %v", err)
	}
	if _, err := CaptureExperimental(context.Background(), Options{}, doc); !errors.Is(err, ErrDisabled) {
		t.Fatalf("opt-in bypassed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CaptureExperimental(ctx, Options{Enabled: true, HelperPath: "unused"}, doc); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
}
