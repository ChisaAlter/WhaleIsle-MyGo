package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestMessageFeedbackAuthoritativeFixture(t *testing.T) {
	var writes atomic.Int32
	var mode atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			RPCID   string
			Payload struct {
				Args struct{ Request map[string]any }
			}
		}
		json.NewDecoder(r.Body).Decode(&q)
		req := q.Payload.Args.Request
		if req["sessionId"] != "s" {
			t.Error("wrong session")
		}
		item := map[string]any{"messageId": "m", "rating": "positive", "note": "my note", "version": "v1"}
		value := any(map[string]any{"ok": true, "value": map[string]any{"items": []any{}}})
		switch r.URL.Path {
		case "/api/messageFeedback/list":
			if mode.Load() == 3 {
				http.NotFound(w, r)
				return
			}
		case "/api/messageFeedback/put":
			writes.Add(1)
			if req["messageId"] != "m" || req["rating"] != "positive" || req["note"] != "my note" {
				t.Error("wrong explicit feedback", req)
			}
			if mode.Load() == 0 && req["ifVersion"] != nil {
				t.Error("create version must be null")
			}
			if mode.Load() == 1 {
				if req["ifVersion"] != "v1" {
					t.Error("edit must retain version")
				}
				item["version"] = "v2"
				value = map[string]any{"ok": false, "error": map[string]any{"code": "version-conflict", "current": item}}
			} else {
				value = map[string]any{"ok": true, "value": item}
			}
		case "/api/messageFeedback/delete":
			writes.Add(1)
			if req["ifVersion"] != "v2" {
				t.Error("delete must use observed version")
			}
			value = map[string]any{"ok": true, "value": map[string]any{"absent": true}}
		default:
			t.Error("unexpected external write", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "s"
	wait := func() {
		t.Helper()
		until := time.Now().Add(time.Second * 3)
		for time.Now().Before(until) {
			v := s.Feedback()
			if !v.Loading && v.BusyID == "" {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("timeout")
	}
	s.RefreshFeedback(context.Background())
	wait()
	if !s.Feedback().Ready {
		t.Fatal(s.Feedback())
	}
	s.BeginFeedback("m", MessageFeedbackUp)
	s.EditFeedback("my note", "")
	if writes.Load() != 0 {
		t.Fatal("opening draft wrote feedback")
	}
	s.SubmitFeedback(context.Background())
	wait()
	if s.Feedback().Items["m"].Version != "v1" {
		t.Fatal(s.Feedback())
	}
	mode.Store(1)
	s.BeginFeedback("m", MessageFeedbackUp)
	s.EditFeedback("my note", "")
	s.SubmitFeedback(context.Background())
	wait()
	if v := s.Feedback(); v.Note != "my note" || v.EditorID != "m" || v.Items["m"].Version != "v2" || v.Error == "" {
		t.Fatal("conflict lost draft/state", v)
	}
	s.CancelFeedback()
	if s.Feedback().Items["m"].Version != "v2" {
		t.Fatal("cancel mutated server state")
	}
	mode.Store(2)
	s.ClearMessageFeedback(context.Background(), "m")
	wait()
	if len(s.Feedback().Items) != 0 {
		t.Fatal("delete not applied")
	}
	mode.Store(3)
	s.RefreshFeedback(context.Background())
	wait()
	before := writes.Load()
	s.RateMessage(context.Background(), "m", MessageFeedbackUp, "")
	wait()
	if s.Feedback().Ready || writes.Load() != before || s.Feedback().Error == "" {
		t.Fatal("missing plugin faked success")
	}
	s.mu.Lock()
	s.clearFeedbackLocked()
	s.mu.Unlock()
	if s.Feedback().Ready || s.Feedback().EditorID != "" {
		t.Fatal("identity reset retained feedback")
	}
}
