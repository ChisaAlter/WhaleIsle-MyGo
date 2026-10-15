package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestSessionFeedbackOnlyExplicitSubmitAndFailureRetainsDraft(t *testing.T) {
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sessionFeedback/record") {
			t.Error("unexpected endpoint", r.URL.Path)
		}
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if count.Add(1) == 1 {
			rpcOK(w, b["rpcId"], map[string]any{"ok": false, "error": map[string]any{"code": "session-not-found"}})
			return
		}
		rpcOK(w, b["rpcId"], map[string]any{"ok": true, "value": map[string]any{"recorded": true}})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.OpenSessionFeedback("s")
	s.EditSessionFeedback(" note ", "other")
	if count.Load() != 0 {
		t.Fatal("opening/editing sent feedback")
	}
	s.SubmitSessionFeedback(context.Background())
	s.SubmitSessionFeedback(context.Background())
	deadline := time.Now().Add(time.Second)
	for s.SessionFeedback().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	v := s.SessionFeedback()
	if !v.Open || v.Text != " note " || v.Error == "" || count.Load() != 1 {
		t.Fatalf("failed feedback lost draft: %+v count %d", v, count.Load())
	}
	s.SubmitSessionFeedback(context.Background())
	deadline = time.Now().Add(time.Second)
	for s.SessionFeedback().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.SessionFeedback().Open || count.Load() != 2 {
		t.Fatal("acknowledgement did not close")
	}
}
func TestSessionFeedbackRejectsUnacknowledgedResult(t *testing.T) {
	for _, raw := range []string{`{}`, `{"ok":true,"value":{}}`, `{"ok":false,"error":{"code":"session-not-found"}}`} {
		if checkSessionFeedbackAck([]byte(raw)) == nil {
			t.Fatal(raw)
		}
	}
}
