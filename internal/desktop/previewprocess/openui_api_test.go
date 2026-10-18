package previewprocess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/desktop/openui"
)

func openUIAPIResponseFixture(id string) openui.APIResponse {
	return openui.APIResponse{InstanceID: openUITestDocument().OpenUI.InstanceID, RequestID: id, Status: 200, Body: json.RawMessage(`{"value":"fixture"}`)}
}
func TestOpenUIAPIReplyAdmissionOwnershipAndBounds(t *testing.T) {
	r := openUIAPIResponseFixture("request")
	var none *Session
	if none.ReplyOpenUIAPI(r) == nil || (&Session{}).ReplyOpenUIAPI(r) == nil {
		t.Fatal("ordinary/nil session accepted response")
	}
	s := openUITestSession()
	defer s.Close()
	if s.ReplyOpenUIAPI(r) == nil {
		t.Fatal("old helper accepted unsupported response")
	}
	s.enableOpenUIAPI(true)
	for _, mutate := range []func(*openui.APIResponse){
		func(r *openui.APIResponse) { r.InstanceID = strings.Repeat("b", 64) }, func(r *openui.APIResponse) { r.RequestID = "" },
		func(r *openui.APIResponse) { r.Status = 199 }, func(r *openui.APIResponse) { r.Status = 600 },
		func(r *openui.APIResponse) { r.Body = json.RawMessage(`{`) }, func(r *openui.APIResponse) {
			r.Body = json.RawMessage(`"` + strings.Repeat("x", maxOpenUIEventBytes) + `"`)
		},
	} {
		bad := r
		mutate(&bad)
		if s.ReplyOpenUIAPI(bad) == nil {
			t.Fatal("invalid response admitted")
		}
	}
	r.RequestID = strings.Repeat("opaque/用户:", 100)
	original := append([]byte(nil), r.Body...)
	if err := s.ReplyOpenUIAPI(r); err != nil {
		t.Fatal("invented requestId restriction", err)
	}
	r.Body[0] = 'x'
	raw := <-s.openUIAPIResponses()
	s.receivedOpenUIAPIResponse(raw)
	var got openui.APIResponse
	if json.Unmarshal(raw, &got) != nil || !bytes.Equal(got.Body, original) {
		t.Fatal("response body alias retained")
	}
	s.Close()
	if s.ReplyOpenUIAPI(openUIAPIResponseFixture("late")) == nil {
		t.Fatal("closed owner accepted response")
	}
}
func TestOpenUIAPIReplyFIFOBackpressureAndUnlimitedLifetime(t *testing.T) {
	s := openUITestSession()
	defer s.Close()
	s.enableOpenUIAPI(true)
	for i := 0; i < maxOpenUIEvents; i++ {
		if err := s.ReplyOpenUIAPI(openUIAPIResponseFixture(fmt.Sprint(i))); err != nil {
			t.Fatal(err)
		}
	}
	blocked := make(chan error, 1)
	go func() { blocked <- s.ReplyOpenUIAPI(openUIAPIResponseFixture("overflow-waits")) }()
	select {
	case err := <-blocked:
		t.Fatal("full response queue did not apply backpressure", err)
	case <-time.After(15 * time.Millisecond):
	}
	for i := 0; i < maxOpenUIEvents; i++ {
		raw := <-s.openUIAPIResponses()
		s.receivedOpenUIAPIResponse(raw)
		var got openui.APIResponse
		_ = json.Unmarshal(raw, &got)
		if got.RequestID != fmt.Sprint(i) {
			t.Fatal("response reordered or coalesced", got.RequestID)
		}
	}
	select {
	case err := <-blocked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("queue space did not wake producer")
	}
	raw := <-s.openUIAPIResponses()
	s.receivedOpenUIAPIResponse(raw)
	for i := 0; i < 100; i++ {
		if err := s.ReplyOpenUIAPI(openUIAPIResponseFixture(fmt.Sprint(i))); err != nil {
			t.Fatal("lifetime request cap", err)
		}
		raw := <-s.openUIAPIResponses()
		s.receivedOpenUIAPIResponse(raw)
	}
	for i := 0; i < maxOpenUIEvents; i++ {
		_ = s.ReplyOpenUIAPI(openUIAPIResponseFixture(fmt.Sprint(i)))
	}
	closed := make(chan error, 1)
	go func() { closed <- s.ReplyOpenUIAPI(openUIAPIResponseFixture("closed")) }()
	s.Close()
	select {
	case err := <-closed:
		if err == nil {
			t.Fatal("closed queue accepted pending reply")
		}
	case <-time.After(time.Second):
		t.Fatal("Close left blocked producer")
	}
	if len(s.openUIAPI.values) != 0 {
		t.Fatal("Close retained buffered responses")
	}
}
func TestOpenUIAPIWireAndFixedShellCall(t *testing.T) {
	r := openUIAPIResponseFixture("opaque")
	r.Body = json.RawMessage(`{"text":"</script>\";window.stolen=true;//","large":"` + strings.Repeat("x", 5000) + `"}`)
	raw, _ := json.Marshal(r)
	if len(raw) <= maxHeaderBytes {
		t.Fatal("fixture misses framed body")
	}
	nonce := strings.Repeat("c", 64)
	var wire bytes.Buffer
	if err := sendOpenUIAPIResponse(&wire, nonce, raw); err != nil {
		t.Fatal(err)
	}
	p, err := readPacket(&wire)
	if err != nil || p.Kind != "api-response" {
		t.Fatal("missing fixed command", err)
	}
	got, err := receiveOpenUIAPIResponse(&wire, p, nonce)
	if err != nil || !bytes.Equal(raw, got) {
		t.Fatal("wire roundtrip", err)
	}
	script, err := openUIAPIResponseScript(got)
	if err != nil || !strings.HasPrefix(script, "(()=>{if(window.__whaleOpenUIAPIResponse({") || !strings.HasSuffix(script, ")!==true)throw new Error('OpenUI API response rejected');return true;})()") || strings.Contains(script, "</script>") {
		t.Fatal("unsafe/flexible shell call", err)
	}
	for _, bad := range []string{string(raw) + ` {}`, strings.TrimSuffix(string(raw), "}") + `,"eval":"evil"}`, `{"instanceId":"bad","requestId":"r","status":200,"body":{}}`, `{"instanceId":"` + r.InstanceID + `","requestId":"r","status":200.5,"body":{}}`} {
		if _, err := openUIAPIResponseScript([]byte(bad)); err == nil {
			t.Fatal("invalid/extra response field accepted")
		}
	}
	p.Nonce = "wrong"
	if _, err := receiveOpenUIAPIResponse(bytes.NewReader(raw), p, nonce); err == nil {
		t.Fatal("wrong protocol nonce accepted")
	}
	p.Nonce = nonce
	p.Bytes = maxOpenUIEventBytes + 1
	if _, err := receiveOpenUIAPIResponse(bytes.NewReader(nil), p, nonce); err == nil {
		t.Fatal("unbounded body allocation")
	}
}
func TestOpenUIAPIPumpFailureClosesOwner(t *testing.T) {
	s := openUITestSession()
	defer s.Close()
	s.enableOpenUIAPI(true)
	_ = s.ReplyOpenUIAPI(openUIAPIResponseFixture("r"))
	startOpenUIAPIPump(context.Background(), s, func([]byte) error { return errors.New("fixture broken pipe") })
	select {
	case <-s.Done():
	case <-time.After(time.Second):
		t.Fatal("response transport failure did not close helper")
	}
}
func TestOpenUIAPINativeSourceAndInlineContract(t *testing.T) {
	raw, err := os.ReadFile("../../../native/macos-preview/main.m")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`@"openUIAPI":@YES`, `response.count!=4`, `size>65536`, `@"apiResponseBody"`, `window.__whaleOpenUIAPIResponse(%@)!==true`, `[status integerValue]<200`, `[status integerValue]>599`, `[self emitOpenUIEvents];[self emitFrame]`} {
		if !bytes.Contains(raw, []byte(fragment)) {
			t.Fatal("native response boundary missing", fragment)
		}
	}
	suspended := true
	p, send := inlinePixelCommand(packet{Kind: "api-response"}, false, &suspended)
	if !send || p.Kind != "api-response" {
		t.Fatal("hidden inline seat dropped response")
	}
}
