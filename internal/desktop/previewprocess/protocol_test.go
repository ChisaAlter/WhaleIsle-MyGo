package previewprocess

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDefaultDisabledBeforePlatformOrDocument(t *testing.T) {
	s, err := OpenExperimental(context.Background(), Options{}, Document{})
	if s != nil || !errors.Is(err, ErrDisabled) {
		t.Fatalf("got %v, %v", s, err)
	}
}

func TestDocumentProtocolRoundTrip(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	doc := Document{HTML: "<script>document.body.textContent='你好 & < >';</script>", Width: 900, Appearance: "dark"}
	var wire bytes.Buffer
	if err := sendDocument(&wire, nonce, doc); err != nil {
		t.Fatal(err)
	}
	got, err := receiveDocument(&wire, nonce)
	if err != nil || got != doc {
		t.Fatalf("got %#v, %v", got, err)
	}
	if wire.Len() != 0 {
		t.Fatal("trailing bytes")
	}
}

func TestPacketBoundsAndStrictFields(t *testing.T) {
	for _, n := range []uint32{0, maxHeaderBytes + 1, 0xffffffff} {
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], n)
		if _, err := readPacket(bytes.NewReader(raw[:])); err == nil {
			t.Fatalf("accepted size %d", n)
		}
	}
	for _, raw := range []string{
		"{",
		`{"version":1,"kind":"ready","nonce":"x","unknown":true}`,
		`{"version":1} {}`,
	} {
		var wire bytes.Buffer
		var size [4]byte
		binary.LittleEndian.PutUint32(size[:], uint32(len(raw)))
		wire.Write(size[:])
		wire.WriteString(raw)
		if _, err := readPacket(&wire); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestDocumentMustFollowExactNonceAndPhase(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	for _, p := range []packet{
		{Version: 1, Kind: "ready", Nonce: nonce, Bytes: 1},
		{Version: 1, Kind: "document", Nonce: strings.Repeat("b", 64), Bytes: 1},
		{Version: 2, Kind: "document", Nonce: nonce, Bytes: 1},
		{Version: 1, Kind: "document", Nonce: nonce, Bytes: maxHTMLBytes + 1},
		{Version: 1, Kind: "document", Nonce: nonce, Bytes: -1},
	} {
		var wire bytes.Buffer
		if err := writePacket(&wire, p); err != nil {
			t.Fatal(err)
		}
		if _, err := receiveDocument(&wire, nonce); err == nil {
			t.Fatalf("accepted %#v", p)
		}
	}
}

func TestTruncatedDocumentFails(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	var wire bytes.Buffer
	if err := writePacket(&wire, packet{Version: 1, Kind: "document", Nonce: nonce, Bytes: 3, Width: 900, Appearance: "dark"}); err != nil {
		t.Fatal(err)
	}
	wire.WriteString("ab")
	if _, err := receiveDocument(&wire, nonce); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v", err)
	}
}

func TestDocumentValidation(t *testing.T) {
	for _, d := range []Document{
		{HTML: " ", Width: 900, Appearance: "dark"},
		{HTML: string([]byte{0xff}), Width: 900, Appearance: "dark"},
		{HTML: "x", Width: 239, Appearance: "dark"},
		{HTML: "x", Width: 1601, Appearance: "dark"},
		{HTML: "x", Width: 900, Appearance: "system"},
	} {
		if err := validateDocument(d); err == nil {
			t.Fatalf("accepted %#v", d)
		}
	}
}

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

func TestZeroProgressWriterFails(t *testing.T) {
	if err := writePacket(zeroWriter{}, packet{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("got %v", err)
	}
}
