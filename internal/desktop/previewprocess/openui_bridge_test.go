package previewprocess

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"

	"whaleisle/internal/desktop/openui"
)

func openUITestDocument() Document {
	return Document{HTML: "<p>Trusted pinned viewer</p>", Width: 300, Appearance: "dark", Inline: true, OpenUI: &openui.Render{InstanceID: strings.Repeat("a", 64), Program: "root = Text(\"hello\")", State: json.RawMessage(`{}`)}}
}
func openUITestEvent(instance, kind string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"channel": "whale-openui", "version": 1, "instanceId": instance, "type": kind, "state": map[string]any{"checked": true}, "message": "user selected"})
	return raw
}
func openUITestSession() *Session {
	s := &Session{done: make(chan struct{}), closeFn: func() error { return nil }}
	s.initOpenUIEvents(openUITestDocument())
	return s
}
func TestOpenUIDocumentTransportUsesTrustedShellAndBoundedBody(t *testing.T) {
	d := openUITestDocument()
	d.OpenUI.Program = strings.Repeat("x", maxHeaderBytes+1)
	var wire bytes.Buffer
	nonce := strings.Repeat("b", 64)
	if err := sendDocument(&wire, nonce, d); err != nil {
		t.Fatal(err)
	}
	got, err := receiveDocument(&wire, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !got.openUIShell || got.OpenUI != nil || !strings.Contains(got.HTML, "__whaleOpenUIDrain") || got.HTML == d.HTML || wire.Len() != 0 {
		t.Fatal("missing prepared OpenUI body/mode")
	}
	if shell, err := documentShell(got); err != nil || shell != got.HTML {
		t.Fatal("trusted shell was wrapped a second time")
	}
	for _, fragment := range []string{"sandbox','allow-scripts allow-forms", "event.source!==frame.contentWindow", "event.origin!=='null'", "connect-src"} {
		if !strings.Contains(got.HTML, fragment) {
			t.Fatal("missing shell boundary", fragment)
		}
	}
	d.capture = true
	if err := sendDocument(io.Discard, nonce, d); err == nil {
		t.Fatal("accepted OpenUI capture")
	}
	d.capture = false
	d.OpenUI.State = json.RawMessage(`null`)
	if err := validateDocument(d); err == nil {
		t.Fatal("accepted invalid state before platform launch")
	}
}
func TestOpenUIDrainBatchBounds(t *testing.T) {
	event := openUITestEvent(strings.Repeat("a", 64), "state")
	valid := "[" + string(event) + "]"
	batch, err := decodeOpenUIBatch(valid)
	if err != nil || len(batch) != 1 {
		t.Fatalf("valid batch: %v", err)
	}
	for _, value := range []any{nil, []any{}, "null", "{}", "[null]", valid + " []", strings.Repeat(" ", maxOpenUIBatchBytes+1), "[" + strings.Repeat(string(event)+",", maxOpenUIEvents) + string(event) + "]", `[{"x":"` + strings.Repeat("x", maxOpenUIEventBytes) + `"}]`} {
		if _, err := decodeOpenUIBatch(value); err == nil {
			t.Fatal("accepted invalid/oversize batch")
		}
	}
	if batch, err := decodeOpenUIBatch("[]"); err != nil || len(batch) != 0 {
		t.Fatal("empty poll failed")
	}
}
func TestOpenUIEventQueueValidationOwnershipAndClose(t *testing.T) {
	s := openUITestSession()
	event := openUITestEvent(strings.Repeat("a", 64), "assistant")
	if (&Session{}).Events() != nil {
		t.Fatal("ordinary HTML exposed event channel")
	}
	for _, bad := range []json.RawMessage{openUITestEvent(strings.Repeat("b", 64), "assistant"), openUITestEvent(strings.Repeat("a", 64), "api-request"), openUITestEvent(strings.Repeat("a", 64), "clipboard"), openUITestEvent(strings.Repeat("a", 64), "open-url"), []byte("{"), make([]byte, maxOpenUIEventBytes+1)} {
		if err := s.acceptOpenUIEvent(bad); err == nil {
			t.Fatal("accepted invalid/stale/privileged event")
		}
	}
	original := append([]byte(nil), event...)
	if err := s.acceptOpenUIEvent(event); err != nil {
		t.Fatal(err)
	}
	event[0] = 'x'
	if got := <-s.Events(); !bytes.Equal(got, original) {
		t.Fatal("event payload alias retained")
	}
	for i := 0; i < maxOpenUIEvents; i++ {
		if err := s.acceptOpenUIEvent(original); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.acceptOpenUIEvent(original); err == nil {
		t.Fatal("unbounded event queue")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := <-s.Events(); ok {
		t.Fatal("stale buffered event survived Close")
	}
	if err := s.acceptOpenUIEvent(original); err == nil {
		t.Fatal("closed queue accepted event")
	}
}
func TestOpenUIEventWireBoundsNonceAndTruncation(t *testing.T) {
	nonce := strings.Repeat("c", 64)
	event := openUITestEvent(strings.Repeat("a", 64), "state")
	var wire bytes.Buffer
	if err := sendOpenUIEvent(&wire, nonce, event); err != nil {
		t.Fatal(err)
	}
	p, err := readPacket(&wire)
	if err != nil {
		t.Fatal(err)
	}
	s := openUITestSession()
	defer s.Close()
	if err := s.receiveOpenUIEvent(&wire, p, nonce); err != nil {
		t.Fatal(err)
	}
	if got := <-s.Events(); !bytes.Equal(got, event) {
		t.Fatal("event wire changed bytes")
	}
	for _, bad := range []packet{{Version: 1, Kind: "openui-event", Nonce: nonce, Bytes: maxOpenUIEventBytes + 1}, {Version: 1, Kind: "openui-event", Nonce: "wrong", Bytes: 1}, {Version: 1, Kind: "theme", Nonce: nonce, Bytes: 1}} {
		if err := s.receiveOpenUIEvent(bytes.NewReader(nil), bad, nonce); err == nil {
			t.Fatal("invalid event header accepted")
		}
	}
	if err := s.receiveOpenUIEvent(bytes.NewReader(nil), p, nonce); err == nil {
		t.Fatal("truncated body accepted")
	}
	if err := (&Session{}).receiveOpenUIEvent(bytes.NewReader(event), p, nonce); err == nil {
		t.Fatal("ordinary HTML accepted event")
	}
}
func TestOpenUIEventsInterleaveOnlyBeforeOneBoundedFrame(t *testing.T) {
	nonce := strings.Repeat("d", 64)
	event := openUITestEvent(strings.Repeat("a", 64), "resolved")
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 300, interactiveHeight))); err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{0, 1, maxOpenUIEvents, maxOpenUIEvents + 1} {
		s := openUITestSession()
		var wire bytes.Buffer
		for i := 0; i < count; i++ {
			if err := sendOpenUIEvent(&wire, nonce, event); err != nil {
				t.Fatal(err)
			}
		}
		if err := sendFrame(&wire, nonce, pngData.Bytes(), 300); err != nil {
			t.Fatal(err)
		}
		got, err := s.receiveOpenUIFrame(&wire, nonce, 300)
		if count > maxOpenUIEvents {
			if err == nil {
				t.Fatal("unbounded events before frame")
			}
		} else if err != nil || !bytes.Equal(got, pngData.Bytes()) || wire.Len() != 0 {
			t.Fatalf("frame/event sequence %d: %v", count, err)
		}
		s.Close()
	}
}
func TestOpenUINativeSourceUsesFixedDrainAndNoCommands(t *testing.T) {
	raw, err := os.ReadFile("../../../native/macos-preview/main.m")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, fragment := range []string{openUIDrainScript, "[batch count]>16", "raw.length>65536", "[self emitOpenUIEvents];[self emitFrame]", "preview.openUI && preview.capture"} {
		if !strings.Contains(source, fragment) {
			t.Fatal("native bridge boundary missing", fragment)
		}
	}
	if strings.Contains(source, "windowScriptObject") || strings.Contains(source, "setWebScriptValue") {
		t.Fatal("page-to-native binding introduced")
	}
}

func TestOpenUIRequiresHelperCapability(t *testing.T) {
	if requireOpenUIBridge(openUITestDocument(), packet{}) == nil {
		t.Fatal("old helper silently accepted OpenUI")
	}
	if requireOpenUIBridge(openUITestDocument(), packet{OpenUI: true}) != nil {
		t.Fatal("capable helper rejected")
	}
	if requireOpenUIBridge(Document{}, packet{}) != nil {
		t.Fatal("ordinary HTML compatibility changed")
	}
}
