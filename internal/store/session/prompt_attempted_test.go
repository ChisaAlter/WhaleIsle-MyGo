package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestPromptAttemptedFailedFirstSubmissionRetainsSessionEngagement(t *testing.T) {
	for _, path := range []string{"echo-abandoned", "prompt-rejected"} {
		t.Run(path, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/session/prompt" {
					t.Errorf("unexpected endpoint %s", r.URL.Path)
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				rpcOK(w, body["rpcId"], map[string]any{"accepted": false})
			}))
			defer srv.Close()
			s := New()
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
			s.items = []Item{{SessionID: "first", Blank: true}, {SessionID: "other", Blank: true}}
			s.selectedID = "first"
			target, err := s.CaptureComposerTarget("first")
			if err != nil {
				t.Fatal(err)
			}
			if path == "echo-abandoned" {
				id := target.BeginSubmission("keep draft", nil)
				if st := s.Snapshot(); !st.PromptAttempted || len(st.Messages) != 1 {
					t.Fatal("submission did not engage on its own frame")
				}
				_ = target.AbandonSubmission(id, errors.New("upload failed"))
			} else if err := target.Submit(context.Background(), "keep draft", nil, 0, ""); err == nil {
				t.Fatal("unaccepted first prompt unexpectedly succeeded")
			}
			if st := s.Snapshot(); !st.PromptAttempted || st.PromptBusy || len(st.Messages) != 0 || st.LastAction == "" || !st.SelectedSummaryBlank {
				t.Fatalf("failed first prompt reverted engagement or hid failure: %+v", st)
			}
			s.mu.Lock()
			s.selectSessionLocked("other")
			s.mu.Unlock()
			if st := s.Snapshot(); st.PromptAttempted {
				t.Fatal("first prompt attempt contaminated another session")
			}
			s.mu.Lock()
			s.selectSessionLocked("first")
			s.mu.Unlock()
			if st := s.Snapshot(); !st.PromptAttempted {
				t.Fatal("resident session lost its prior attempt on navigation")
			}
			s.Attach(&dsh.Client{}, nil)
			if st := s.Snapshot(); st.PromptAttempted {
				t.Fatal("new Host inherited another Host's attempt")
			}
		})
	}
}
