package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"whaleisle/internal/dsh"
)

func TestComposerRejectsUnacceptedReceipt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		rpcOK(w, b["rpcId"], map[string]any{"accepted": false})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "a"
	if err := s.SubmitComposer(context.Background(), "a", "keep me", nil, 0); err == nil {
		t.Fatal("unaccepted receipt cleared the draft")
	}
}

func TestComposerIndependentAdmissionsAndEchoIdentity(t *testing.T) {
	entered := make(chan string, 2)
	release := map[string]chan struct{}{"one": make(chan struct{}), "two": make(chan struct{})}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args struct {
					Request struct {
						RequestID, SessionID, Mode string
						Content                    []struct {
							Text string `json:"text"`
						} `json:"content"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		req := b.Payload.Args.Request
		msg := req.Content[0].Text
		if req.SessionID != "a" || req.Mode != "queue" || req.RequestID == "" {
			t.Errorf("unbound request: %+v", req)
		}
		entered <- msg
		<-release[msg]
		rpcOK(w, b.RPCID, map[string]any{"accepted": true})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "a"
	a, _ := s.CaptureComposerTarget("a")
	b, _ := s.CaptureComposerTarget("a")
	aid, bid := a.BeginSubmission("one", nil), b.BeginSubmission("two", nil)
	if aid == bid || len(s.Snapshot().Messages) != 2 {
		t.Fatal("independent echo identities missing")
	}
	done1, done2 := make(chan error, 1), make(chan error, 1)
	go func() { done1 <- a.Submit(context.Background(), "one", nil, 0, aid) }()
	go func() { done2 <- b.Submit(context.Background(), "two", nil, 0, bid) }()
	<-entered
	<-entered
	if !s.Snapshot().PromptBusy {
		t.Fatal("admission not busy")
	}
	close(release["one"])
	if err := <-done1; err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if !snap.PromptBusy || len(snap.Messages) != 2 || !snap.Messages[0].Submission.Accepted {
		t.Fatal("first receipt locked or erased another send")
	}
	s.applyPersistenceEvent(json.RawMessage(`{"type":"user/message","seq":1,"data":{"source":{"kind":"user","rpcId":"wrong"},"content":[{"type":"text","text":"one"}]}}`))
	if len(s.composerPending) != 2 {
		t.Fatal("echo retired by text")
	}
	queue, _ := json.Marshal(map[string]any{"next-turn": []any{map[string]any{"id": "q", "source": map[string]any{"kind": "user", "rpcId": aid}, "content": []any{map[string]any{"type": "text", "text": "one"}}}}})
	s.mu.Lock()
	s.applyQueueInboxLocked(queue, 2)
	s.mu.Unlock()
	if len(s.Snapshot().PlanBoard.Queue) != 0 || len(s.composerPending) != 2 {
		t.Fatal("idle transcript echo duplicated in queue")
	}
	raw, _ := json.Marshal(map[string]any{"type": "user/message", "seq": 3, "data": map[string]any{"source": map[string]any{"kind": "user", "rpcId": aid}, "content": []any{map[string]any{"type": "text", "text": "one"}}}})
	s.applyPersistenceEvent(raw)
	if len(s.composerPending) != 1 {
		t.Fatal("durable identity did not retire one echo")
	}
	close(release["two"])
	if err := <-done2; err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().PromptBusy || len(s.composerPending) != 1 {
		t.Fatal("receipt not separated from observation")
	}
}

func TestComposerQueuedObservationAndHostIsolation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		rpcOK(w, b["rpcId"], map[string]any{"accepted": false})
	}))
	defer srv.Close()
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.client = client
	s.selectedID = "a"
	s.items = []Item{{SessionID: "a", Running: true}}
	owner, _ := s.CaptureComposerTarget("a")
	rid := owner.BeginSubmission("queued", nil)
	if len(s.Snapshot().PlanBoard.Pending) != 1 || len(s.Snapshot().Messages) != 0 {
		t.Fatal("wrong queue placement")
	}
	raw, _ := json.Marshal(map[string]any{"next-turn": []any{map[string]any{"id": "q", "source": map[string]any{"kind": "user", "rpcId": rid}, "content": []any{map[string]any{"type": "text", "text": "queued"}}}}})
	s.mu.Lock()
	s.applyQueueInboxLocked(raw, 1)
	s.mu.Unlock()
	if len(s.Snapshot().PlanBoard.Pending) != 0 || len(s.Snapshot().PlanBoard.Queue) != 1 {
		t.Fatal("queue echo handoff duplicated/lost")
	}
	if err := owner.AbandonSubmission(rid, context.DeadlineExceeded); err != nil {
		t.Fatal("observed admission must win late transport error")
	}
	s.selectedID = "b"
	s.fire()
	other, _ := s.CaptureComposerTarget("a")
	id := other.BeginSubmission("for a", nil)
	if len(s.Snapshot().Messages) != 0 {
		t.Fatal("echo leaked into b")
	}
	s.selectedID = "a"
	s.fire()
	if len(s.Snapshot().PlanBoard.Pending) != 1 {
		t.Fatal("owner echo lost on navigation")
	}
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	if len(s.Snapshot().Messages) != 0 || len(s.Snapshot().PlanBoard.Pending) != 0 {
		t.Fatal("old Host echo survived replacement")
	}
	if err := other.Submit(context.Background(), "for a", nil, 0, id); err == nil {
		t.Fatal("old Host flight sent through replacement")
	}
}

func TestComposerModeCapturedBeforeAdmission(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		got = b["payload"].(map[string]any)["args"].(map[string]any)["request"].(map[string]any)["mode"].(string)
		rpcOK(w, b["rpcId"], map[string]any{"accepted": true})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "a"
	owner, _ := s.CaptureComposerTarget("a")
	s.SetPromptMode(PromptModeSteer)
	if err := owner.Submit(context.Background(), "captured", nil, 0, ""); err != nil || got != "queue" {
		t.Fatalf("mode %s: %v", got, err)
	}
}
