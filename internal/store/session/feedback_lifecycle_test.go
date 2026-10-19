package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
)

type feedbackRoundTrip func(*http.Request) (*http.Response, error)

func (f feedbackRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFeedbackDraftSettlementOwnership(t *testing.T) {
	for _, outcome := range []string{"success", "failure", "conflict"} {
		for _, action := range []string{"keep", "dismiss", "reopen-same", "reopen-other", "selection"} {
			t.Run(outcome+"/"+action, func(t *testing.T) {
				entered, release := make(chan struct{}), make(chan struct{})
				requestContext := make(chan (<-chan struct{}), 1)
				var releaseOnce sync.Once
				var writes atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var q struct {
						RPCID   string
						Payload struct {
							Args struct{ Request map[string]any }
						}
					}
					_ = json.NewDecoder(r.Body).Decode(&q)
					if r.URL.Path != "/api/messageFeedback/put" || q.Payload.Args.Request["sessionId"] != "s" || q.Payload.Args.Request["note"] != "submitted note" {
						t.Errorf("wrong feedback action: %s %+v", r.URL.Path, q)
					}
					writes.Add(1)
					close(entered)
					<-release
					item := map[string]any{"messageId": "m", "rating": MessageFeedbackUp, "version": "v1"}
					value := any(map[string]any{"ok": true, "value": item})
					if outcome != "success" {
						code := "note-too-large"
						if outcome == "conflict" {
							code = "version-conflict"
						}
						value = map[string]any{"ok": false, "error": map[string]any{"code": code, "current": item}}
					}
					rpcOK(w, q.RPCID, value)
				}))
				defer srv.Close()
				defer releaseOnce.Do(func() { close(release) })
				s := New()
				s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: &http.Client{Transport: feedbackRoundTrip(func(r *http.Request) (*http.Response, error) {
					requestContext <- r.Context().Done()
					return http.DefaultTransport.RoundTrip(r)
				})}}
				s.selectedID = "s"
				s.feedback.Ready = true
				s.feedback.Items = map[string]FeedbackItem{}
				s.messageRatings = map[string]string{}
				s.BeginFeedback("m", MessageFeedbackUp)
				s.EditFeedback("submitted note", "bug")
				s.SubmitFeedback(context.Background())
				wait(t, entered)
				completed := <-requestContext
				switch action {
				case "dismiss", "reopen-same", "reopen-other":
					s.CancelFeedback()
					if v := s.Feedback(); v.EditorID != "" || v.Note != "" || v.Category != "" || v.Error != "" || v.BusyID != "m" || s.Snapshot().FeedbackBusyID != "m" {
						t.Fatalf("dismiss erased request identity or retained draft: %+v", v)
					}
					if action == "reopen-same" {
						s.BeginFeedback("m", MessageFeedbackDown)
					} else if action == "reopen-other" {
						s.BeginFeedback("next", MessageFeedbackDown)
					}
				case "selection":
					s.mu.Lock()
					s.selectSessionLocked("other")
					s.feedback.Ready = true
					s.feedback.Items = map[string]FeedbackItem{}
					s.mu.Unlock()
					s.BeginFeedback("next", MessageFeedbackDown)
					s.EditFeedback("new session draft", "other")
				}
				before := s.Feedback()
				// The originating request remains in flight, so a second submit
				// cannot discard its BusyID or send a duplicate command.
				if action != "selection" {
					s.SubmitFeedback(context.Background())
				}
				releaseOnce.Do(func() { close(release) })
				// mutateFeedback cancels its derived request context on return,
				// after the ownership fold; this observes settlement without sleep.
				wait(t, completed)
				v := s.Feedback()
				if writes.Load() != 1 {
					t.Fatalf("dismiss/reopen submitted %d writes", writes.Load())
				}
				if action == "selection" {
					if !reflect.DeepEqual(v, before) || s.Snapshot().FeedbackBusyID != "" {
						t.Fatalf("old response changed new session: %+v", v)
					}
					return
				}
				if v.BusyID != "" || s.Snapshot().FeedbackBusyID != "" {
					t.Fatal("settlement did not release request identity")
				}
				if action != "keep" && (v.EditorID != before.EditorID || v.Rating != before.Rating || v.Note != before.Note || v.Category != before.Category || v.Error != "") {
					t.Fatalf("late response changed dismissed/replacement draft: before=%+v after=%+v", before, v)
				}
				if outcome == "success" {
					if v.Acknowledgement != 1 || v.Items["m"].Version != "v1" || s.Snapshot().LastAction != "feedback saved" || action == "keep" && v.EditorID != "" {
						t.Fatalf("successful action lost commit or acknowledgement: %+v", v)
					}
				} else {
					if v.Acknowledgement != 0 || action == "keep" && (v.Error == "" || v.Note != "submitted note" || v.EditorID != "m") {
						t.Fatalf("failed action lost current draft or acknowledged success: %+v", v)
					}
					if outcome == "conflict" && v.Items["m"].Version != "v1" {
						t.Fatal("dismissed conflict lost authoritative item")
					}
				}
			})
		}
	}
}

func TestFeedbackRepeatedIdenticalSuccessAcknowledgesEveryWrite(t *testing.T) {
	requestContext := make(chan (<-chan struct{}), 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ RPCID string }
		_ = json.NewDecoder(r.Body).Decode(&q)
		rpcOK(w, q.RPCID, map[string]any{"ok": true, "value": map[string]any{"messageId": "m", "rating": MessageFeedbackUp, "version": "same"}})
	}))
	defer srv.Close()
	s := New()
	s.selectedID = "s"
	s.feedback.Ready = true
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: &http.Client{Transport: feedbackRoundTrip(func(r *http.Request) (*http.Response, error) {
		requestContext <- r.Context().Done()
		return http.DefaultTransport.RoundTrip(r)
	})}}
	for i := uint64(1); i <= 2; i++ {
		s.BeginFeedback("m", MessageFeedbackUp)
		s.SubmitFeedback(context.Background())
		wait(t, <-requestContext)
		if v := s.Feedback(); v.Acknowledgement != i || v.Items["m"].Version != "same" || v.EditorID != "" {
			t.Fatalf("identical successful reply lost acknowledgement %d: %+v", i, v)
		}
	}
}
