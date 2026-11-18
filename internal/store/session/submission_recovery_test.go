package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestSubmissionRecoveryWritesBeforeDispatchAndKeepsUnknownOutcome(t *testing.T) {
	s := newRetirementRegressionStore()
	var phases []string
	var requests int
	s.client.HTTP.Transport = composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		if len(phases) == 0 || phases[len(phases)-1] != "dispatching" {
			t.Fatal("request reached transport before its recovery record")
		}
		return nil, io.ErrUnexpectedEOF
	})
	if err := s.BindSubmissionRecovery(context.Background(), "test-profile", nil, func(record SubmissionRecord) error {
		phases = append(phases, record.Phase)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	owner, _ := s.CaptureComposerTarget("a")
	id := owner.BeginSubmission("preserve me", nil)
	if err := owner.StageSubmission(id); err != nil {
		t.Fatal(err)
	}
	err := owner.Submit(context.Background(), "preserve me", nil, 0, id)
	if !errors.Is(err, ErrSubmissionUnconfirmed) || requests != 1 || s.composerPending[id] == nil || !s.composerPending[id].Unconfirmed {
		t.Fatalf("unknown admission became a retryable failure: err=%v calls=%d pending=%+v", err, requests, s.composerPending[id])
	}
}

func TestSubmissionRecoveryWriteFailurePreventsDispatch(t *testing.T) {
	s := newRetirementRegressionStore()
	calls := 0
	s.client.HTTP.Transport = composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return composerRegressionReply(r, map[string]bool{"accepted": true})
	})
	s.BindSubmissionRecovery(context.Background(), "profile", nil, func(SubmissionRecord) error { return errors.New("disk full") })
	owner, _ := s.CaptureComposerTarget("a")
	id := owner.BeginSubmission("retain", nil)
	if err := owner.Submit(context.Background(), "retain", nil, 0, id); err == nil || calls != 0 {
		t.Fatalf("dispatch escaped failed write: %v calls=%d", err, calls)
	} else if errors.Is(err, ErrSubmissionUnconfirmed) {
		t.Fatalf("pre-dispatch failure was incorrectly made uncertain: %v", err)
	}
}

func TestSubmissionConfirmationUsesRequestSessionAfterNavigation(t *testing.T) {
	s := newRetirementRegressionStore()
	var terminal SubmissionRecord
	s.BindSubmissionRecovery(context.Background(), "profile", nil, func(record SubmissionRecord) error { terminal = record; return nil })
	owner, id := beginAcceptedRetirement(t, s, "a")
	p := *s.composerPending[id]
	s.selectedID = "b"
	raw, _ := json.Marshal(map[string]any{"type": "user/message", "seq": 11, "data": map[string]any{"source": map[string]string{"kind": "user", "rpcId": id}}})
	s.mu.Lock()
	s.observeComposerEventForSessionLocked("b", raw)
	s.mu.Unlock()
	assertRetirementPending(t, owner, id)
	s.mu.Lock()
	s.observeComposerEventForSessionLocked("a", raw)
	s.mu.Unlock()
	if err := awaitRetirementRegression(t, owner, id); err != nil {
		t.Fatal(err)
	}
	if err := s.persistSubmissionOutcome(&p, nil); err != nil {
		t.Fatal(err)
	}
	if terminal.Phase != "confirmed" || s.selectedID != "b" {
		t.Fatalf("wrong confirmation owner: %+v %s", terminal, s.selectedID)
	}
}

func TestSubmissionRecoveryRestoresIdentityWithoutDispatch(t *testing.T) {
	s := newRetirementRegressionStore()
	requests := 0
	s.client.HTTP.Transport = composerRegressionTransport(func(r *http.Request) (*http.Response, error) { requests++; return nil, errors.New("must not dispatch") })
	record := SubmissionRecord{RequestID: "stable", SessionID: "a", Text: "retained", Phase: "dispatching", Placement: "transcript"}
	raw, _ := json.Marshal(SubmissionRecovery{Version: 1, Owner: "profile", Records: map[string]SubmissionRecord{"stable": record}})
	if err := s.BindSubmissionRecovery(context.Background(), "profile", raw, func(SubmissionRecord) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if requests != 0 || s.composerPending["stable"] == nil || !s.composerPending["stable"].Unconfirmed {
		t.Fatal("restored request lost identity or was replayed")
	}
	record.Phase = "prepared"
	raw, _ = json.Marshal(SubmissionRecovery{Version: 1, Owner: "profile", Records: map[string]SubmissionRecord{"prepared": {RequestID: "prepared", SessionID: "a", Text: "not sent", Phase: "prepared"}}})
	if err := s.BindSubmissionRecovery(context.Background(), "profile", raw, func(r SubmissionRecord) error {
		if r.Phase != "failed" {
			t.Fatal(r)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err, ok := s.SubmissionOutcome("prepared"); !ok || err == nil {
		t.Fatal("pre-dispatch interruption did not return the draft")
	}
}

func TestSubmissionHostChangeDoesNotMakeDispatchSafeToRetry(t *testing.T) {
	s := newRetirementRegressionStore()
	owner, _ := s.CaptureComposerTarget("a")
	id := owner.BeginSubmission("pending", nil)
	s.client.HTTP.Transport = composerRegressionTransport(func(r *http.Request) (*http.Response, error) { s.Detach(); return nil, io.EOF })
	if err := owner.Submit(context.Background(), "pending", nil, 0, id); !errors.Is(err, ErrSubmissionUnconfirmed) {
		t.Fatalf("lost dispatch certainty after detach: %v", err)
	}
}

func TestCommandSubmissionPersistsOnlyActualResult(t *testing.T) {
	for _, result := range []string{"success", "error", "transport"} {
		t.Run(result, func(t *testing.T) {
			s := newRetirementRegressionStore()
			var mu sync.Mutex
			var phases []string
			s.BindSubmissionRecovery(context.Background(), "profile", nil, func(record SubmissionRecord) error {
				mu.Lock()
				defer mu.Unlock()
				phases = append(phases, record.Phase)
				return nil
			})
			s.client.HTTP.Transport = composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/commands/list") {
					return composerRegressionReply(r, []any{map[string]any{"name": "review"}})
				}
				mu.Lock()
				for _, phase := range phases {
					if phase == "confirmed" || phase == "failed" {
						t.Error("command retired before its execution result")
					}
				}
				mu.Unlock()
				if result == "transport" {
					return nil, io.ErrUnexpectedEOF
				}
				return composerRegressionReply(r, map[string]any{"result": map[string]any{"kind": result, "text": "command failure"}})
			})
			owner, _ := s.CaptureComposerTarget("a")
			id := owner.BeginSubmission("/review", nil)
			err := owner.Submit(context.Background(), "/review", nil, 0, id)
			if result == "transport" {
				if !errors.Is(err, ErrSubmissionUnconfirmed) {
					t.Fatal(err)
				}
				if _, known := s.SubmissionOutcome(id); known {
					t.Fatal("lost command response became a terminal result")
				}
			} else {
				if errors.Is(err, ErrSubmissionUnconfirmed) || (result == "success") != (err == nil) {
					t.Fatal(err)
				}
				if _, known := s.SubmissionOutcome(id); !known {
					t.Fatal("actual command result did not reach its draft owner")
				}
			}
		})
	}
}
