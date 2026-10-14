package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"whaleisle/internal/dsh"
)

func TestMessageEditFixture(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args struct {
					Request struct {
						SessionID      string  `json:"sessionId"`
						EditMessageSeq float64 `json:"editMessageSeq"`
						Content        []struct {
							Text string `json:"text"`
						} `json:"content"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		req := body.Payload.Args.Request
		if r.URL.Path != "/api/rpc/session/prompt" && r.URL.Path != "/rpc/session/prompt" {
			t.Log("prompt endpoint", r.URL.Path)
		}
		if req.SessionID != "s" || req.EditMessageSeq != 2 || len(req.Content) != 1 || req.Content[0].Text != "/new" {
			t.Errorf("edit sent as command/wrong target: %+v", req)
		}
		rpcOK(w, body.RPCID, map[string]any{})
	}))
	defer srv.Close()
	st := New()
	st.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.selectedID = "s"
	apply := func(raw string) { st.applyPersistenceEvent(json.RawMessage(raw)) }
	apply(`{"type":"turn/start","seq":1,"time":1,"data":{"turn":0}}`)
	apply(`{"type":"user/message","seq":2,"surfaceOp":"append","data":{"source":{"kind":"user"},"content":[{"type":"text","text":"old"}]}}`)
	apply(`{"type":"user/message","seq":3,"surfaceOp":"append","data":{"source":{"kind":"plugin"},"content":[{"type":"text","text":"injected"}]}}`)
	apply(`{"type":"user/message","seq":4,"surfaceOp":"append","data":{"source":{"kind":"user"},"content":[{"type":"text","text":"steering"}]}}`)
	if CanEditMessage(st.Snapshot(), 2) {
		t.Fatal("running edit allowed")
	}
	apply(`{"type":"turn/end","seq":5,"time":2,"data":{"turn":0}}`)
	if !CanEditMessage(st.Snapshot(), 2) || CanEditMessage(st.Snapshot(), 4) {
		t.Fatal("latest turn-opening guard")
	}
	if err := st.SubmitComposer(context.Background(), "s", "/new", nil, 2); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	apply(`{"type":"turn/start","seq":6,"time":3,"data":{"turn":1}}`)
	apply(`{"type":"user/message","seq":7,"surfaceOp":{"op":"replace","startSeq":2,"endSeq":4},"data":{"source":{"kind":"user","edit":{"turn":0,"messageSeq":2}},"content":[{"type":"text","text":"new"}]}}`)
	apply(`{"type":"turn/end","seq":8,"time":4,"data":{"turn":1}}`)
	snap := st.Snapshot()
	if len(snap.Messages) != 2 || snap.Messages[0].Body != "new" || len(st.messages) != 6 {
		t.Fatalf("replacement projection/log retention: %+v journal=%d", snap.Messages, len(st.messages))
	}
	if err := st.SubmitComposer(context.Background(), "s", "stale", nil, 2); err == nil || calls != 1 {
		t.Fatal("stale dispatch")
	}
	st.messages[4].Raw = json.RawMessage(`{"type":"user/message","data":{"source":{"kind":"user"},"content":[{"type":"image"}]}}`)
	if CanEditMessage(st.Snapshot(), 7) {
		t.Fatal("multimodal edit allowed")
	}
}
