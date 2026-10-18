package previewprocess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/desktop/openui"
)

func TestOpenUIUpdateLatestOnlyRateLimitAndStatePreservation(t *testing.T) {
	s := openUITestSession()
	defer s.Close()
	s.enableOpenUIUpdates(true)
	r := *openUITestDocument().OpenUI
	r.State = json.RawMessage(`{"stale":true}`)
	for _, program := range []string{"first", "second", "latest"} {
		r.Program = program
		if err := s.UpdateOpenUI(r); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Unix(100, 0)
	raw := s.takeOpenUIUpdate(now)
	var got map[string]json.RawMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || string(got["program"]) != `"latest"` || got["state"] != nil || got["theme"] != nil {
		t.Fatalf("update did not coalesce/retain state ownership: %s", raw)
	}
	r.Program = "after"
	r.Streaming = true
	if err := s.UpdateOpenUI(r); err != nil {
		t.Fatal(err)
	}
	if s.takeOpenUIUpdate(now.Add(99*time.Millisecond)) != nil {
		t.Fatal("exceeded 10 Hz")
	}
	next := s.takeOpenUIUpdate(now.Add(100 * time.Millisecond))
	if !bytes.Contains(next, []byte(`"program":"after"`)) || !bytes.Contains(next, []byte(`"streaming":true`)) {
		t.Fatal("latest streaming update lost")
	}
	if err := s.UpdateOpenUI(r); err != nil {
		t.Fatal(err)
	}
	if s.takeOpenUIUpdate(now.Add(time.Second)) != nil {
		t.Fatal("duplicate update retransmitted")
	}
	r.Program = "pending"
	if err := s.UpdateOpenUI(r); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if s.takeOpenUIUpdate(now.Add(time.Second)) != nil || s.UpdateOpenUI(r) == nil {
		t.Fatal("closed session retained/admitted update")
	}
}
func TestOpenUIUpdateRejectsWrongModeInstanceAndBounds(t *testing.T) {
	r := *openUITestDocument().OpenUI
	if (&Session{}).UpdateOpenUI(r) == nil {
		t.Fatal("ordinary HTML accepted update")
	}
	var nilSession *Session
	if nilSession.UpdateOpenUI(r) == nil {
		t.Fatal("nil session accepted update")
	}
	s := openUITestSession()
	defer s.Close()
	if s.UpdateOpenUI(r) == nil {
		t.Fatal("old helper accepted updates")
	}
	s.enableOpenUIUpdates(true)
	for _, mutate := range []func(*openui.Render){func(r *openui.Render) { r.InstanceID = strings.Repeat("b", 64) }, func(r *openui.Render) { r.Program = strings.Repeat("x", openui.MaxProgramBytes+1) }, func(r *openui.Render) {
		r.State = json.RawMessage(`{"x":"` + strings.Repeat("x", openui.MaxStateBytes) + `"}`)
	}} {
		bad := r
		mutate(&bad)
		if s.UpdateOpenUI(bad) == nil {
			t.Fatal("invalid update accepted")
		}
	}
}
func TestOpenUIUpdateWireAndFixedScript(t *testing.T) {
	r := *openUITestDocument().OpenUI
	r.Program = `</script>";window.stolen=true;//` + strings.Repeat("<", openui.MaxProgramBytes-40)
	raw, err := json.Marshal(openUIUpdate{r.InstanceID, r.Program, true})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) <= maxHeaderBytes || len(raw) > maxOpenUIRenderBytes {
		t.Fatal("fixture does not exercise escaped body bounds")
	}
	nonce := strings.Repeat("a", 64)
	var wire bytes.Buffer
	if err := sendOpenUIUpdate(&wire, nonce, raw); err != nil {
		t.Fatal(err)
	}
	p, err := readPacket(&wire)
	if err != nil {
		t.Fatal(err)
	}
	result, err := receiveOpenUIUpdate(&wire, p, nonce)
	if err != nil {
		t.Fatal(err)
	}
	script, err := openUIUpdateScript(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(script, "window.__whaleOpenUIRender({") || !strings.HasSuffix(script, ");true") || strings.Contains(script, "</script>") || strings.Contains(script, `"state"`) {
		t.Fatal("unsafe or state-overwriting script")
	}
	for _, bad := range []string{`{"instanceId":"` + r.InstanceID + `","program":"x","streaming":false,"state":{}}`, string(raw) + ` {}`, `{"instanceId":"` + r.InstanceID + `","program":"x","streaming":"true"}`} {
		if _, err := validateOpenUIUpdate([]byte(bad)); err == nil {
			t.Fatal("unknown/trailing/untyped render accepted")
		}
	}
	if _, err := receiveOpenUIUpdate(bytes.NewReader(nil), p, nonce); !errors.Is(err, io.EOF) {
		t.Fatalf("missing body: %v", err)
	}
	p.Bytes = maxOpenUIRenderBytes + 1
	if _, err := receiveOpenUIUpdate(bytes.NewReader(nil), p, nonce); err == nil {
		t.Fatal("oversized body admitted")
	}
	p.Bytes = 1
	p.Nonce = "wrong"
	if _, err := receiveOpenUIUpdate(bytes.NewReader([]byte("x")), p, nonce); err == nil {
		t.Fatal("wrong nonce admitted")
	}
}
func TestOpenUIUpdatePumpClosesOnTransportFailure(t *testing.T) {
	s := openUITestSession()
	s.enableOpenUIUpdates(true)
	r := *openUITestDocument().OpenUI
	r.Program = "new"
	if err := s.UpdateOpenUI(r); err != nil {
		t.Fatal(err)
	}
	startOpenUIUpdatePump(context.Background(), s, func([]byte) error { return errors.New("pipe closed") })
	select {
	case <-s.Done():
	case <-time.After(time.Second):
		t.Fatal("failed update transport retained helper")
	}
}
func TestOpenUIUpdateMacSourceAndInlineContract(t *testing.T) {
	raw, err := os.ReadFile("../../../native/macos-preview/main.m")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`@"openUIUpdate":@YES`, `size>2*1024*1024`, `window.__whaleOpenUIRender(%@);true`, `next.count!=3`, `program lengthOfBytesUsingEncoding:NSUTF8StringEncoding]>262144`, `[self emitOpenUIEvents];[self emitFrame]`} {
		if !bytes.Contains(raw, []byte(fragment)) {
			t.Fatal("native render boundary missing", fragment)
		}
	}
	suspended := true
	command, send := inlinePixelCommand(packet{Kind: "render"}, false, &suspended)
	if !send || command.Kind != "render" {
		t.Fatal("render data incorrectly treated as stale input")
	}
}
