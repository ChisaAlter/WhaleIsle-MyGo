package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
)

// One bounded mock fixture covers publication authority, exact wire identity,
// complete source, old Host, cross-session and late completion behavior.
func TestVisualReplyReaderFixture(t *testing.T) {
	html := "<!doctype html><script>fetch('https://example.invalid')</script>\n" + strings.Repeat("完整源码", 750000)
	ref := VisualReplyFile{AttachmentID: "page-1", Name: "page.html", Bytes: int64(len(html))}
	native, _ := json.Marshal(map[string]any{"type": "tool/result", "surfaceOp": "append", "data": map[string]any{"message": map[string]any{"content": []any{map[string]any{"type": "file", "attachment": ref}}}, "meta": map[string]any{"visualReply": map[string]any{"version": 1, "title": "Page"}}}})
	ptc, _ := json.Marshal(map[string]any{"type": "tool/ptc-dispatch", "data": map[string]any{"name": "html_render", "arguments": map[string]any{"title": "PTC Page"}, "content": []any{map[string]any{"type": "file", "attachment": ref}}}})
	for _, raw := range []json.RawMessage{native, ptc} {
		if p, ok := PublishedVisualReply(raw); !ok || p.File != ref {
			t.Fatalf("publication not preserved: %s", raw)
		}
	}
	for _, raw := range []string{strings.Replace(string(native), `"append"`, `"replace"`, 1), strings.Replace(string(native), `"version":1`, `"version":2`, 1), strings.Replace(string(native), `"message":{`, `"message":{"isError":true,`, 1), strings.Replace(string(ptc), `"html_render"`, `"write"`, 1), strings.Replace(string(ptc), `"bytes":`, `"bytes":0,"bad":`, 1)} {
		if _, ok := PublishedVisualReply(json.RawMessage(raw)); ok {
			t.Fatal("accepted nonpublication", raw)
		}
	}
	var mode atomic.Int32
	var calls atomic.Int32
	entered, release := make(chan struct{}, 1), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/session/readVisualReply" {
			t.Error("unauthorized fallback", r.URL.Path)
			http.Error(w, "unexpected", 500)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		req := body["payload"].(map[string]any)["args"].(map[string]any)["request"].(map[string]any)
		if len(req) != 2 || req["sessionId"] != "A" || req["attachmentId"] != "page-1" {
			t.Error("wrong read identity", req)
		}
		switch mode.Load() {
		case 1:
			http.Error(w, "method not found", 404)
			return
		case 2:
			entered <- struct{}{}
			<-release
		case 3:
			rpcOK(w, body["rpcId"], map[string]any{"attachment": ref, "html": "truncated"})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{"attachment": ref, "html": html})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.selectedID = "A"
	s.messages = []Message{{Raw: native}}
	s.LoadVisualReply(context.Background(), "B", "page-1")
	s.LoadVisualReply(context.Background(), "A", "unknown")
	if calls.Load() != 0 {
		t.Fatal("cross-session/unpublished read escaped")
	}
	s.LoadVisualReply(context.Background(), "A", "page-1")
	if v := s.VisualReplySnapshot(); !v.Ready || v.HTML != html {
		t.Fatalf("full source missing: %+v", v)
	}
	mode.Store(1)
	s.LoadVisualReply(context.Background(), "A", "page-1")
	if v := s.VisualReplySnapshot(); v.Ready || v.HTML != "" || !strings.Contains(v.Error, "404") {
		t.Fatal("old Host did not fail closed", v)
	}
	mode.Store(3)
	s.LoadVisualReply(context.Background(), "A", "page-1")
	if v := s.VisualReplySnapshot(); v.Ready || v.HTML != "" || v.Error == "" {
		t.Fatal("partial source accepted", v)
	}
	mode.Store(2)
	done := make(chan struct{})
	go func() { s.LoadVisualReply(context.Background(), "A", "page-1"); close(done) }()
	<-entered
	s.mu.Lock()
	s.selectedID = "B"
	s.followGeneration++
	s.mu.Unlock()
	if v := s.VisualReplySnapshot(); v.AttachmentID != "" {
		t.Fatal("old session source leaked", v)
	}
	close(release)
	<-done
	if v := s.VisualReplySnapshot(); v.AttachmentID != "" {
		t.Fatal("late reply leaked", v)
	}
	mode.Store(0)
	s.mu.Lock()
	s.selectedID = "A"
	s.mu.Unlock()
	s.LoadVisualReply(context.Background(), "A", "page-1")
	s.CloseVisualReply()
	if v := s.VisualReplySnapshot(); v.Ready || v.HTML != "" {
		t.Fatal("close retained visible source")
	}
	if calls.Load() != 5 {
		t.Fatal("unexpected RPC count", calls.Load())
	}
}
