package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// The production client still serializes and decodes the real RPC envelope.
// This transport opens no socket and cannot dispatch to an actual Host/model.
type composerRegressionTransport func(*http.Request) (*http.Response, error)

func (f composerRegressionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func composerRegressionReply(r *http.Request, value any) (*http.Response, error) {
	var request struct {
		RPCID string `json:"rpcId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
}

func newRetirementRegressionStore() *Store {
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://retirement.invalid"}, HTTP: &http.Client{Transport: composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/session/prompt") {
			return nil, fmt.Errorf("unexpected method %s", r.URL.Path)
		}
		return composerRegressionReply(r, map[string]bool{"accepted": true})
	})}}
	s.selectedID = "a"
	s.items = []Item{{SessionID: "a", Running: true}, {SessionID: "b", Running: true}}
	s.promptMode = PromptModeSteer
	return s
}

func beginAcceptedRetirement(t *testing.T, s *Store, sessionID string) (*ComposerTarget, string) {
	t.Helper()
	owner, err := s.CaptureComposerTarget(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	attachments := []ComposerAttachment{{Path: sessionID + ".png", Name: sessionID + ".png", ImageData: "cG5n", MediaType: "image/png", SessionID: sessionID, HostScope: owner.scope()}}
	id := owner.BeginSubmission("retained "+sessionID, attachments)
	if id == "" {
		t.Fatal("submission did not acquire its owner")
	}
	if err := owner.Submit(context.Background(), "retained "+sessionID, attachments, 0, id); err != nil {
		t.Fatal(err)
	}
	return owner, id
}

func assertRetirementPending(t *testing.T, owner *ComposerTarget, id string) {
	t.Helper()
	select {
	case <-owner.retirements[id].done:
		t.Fatal("RPC acceptance or an unrelated lifecycle event retired the attachment flight")
	default:
	}
}

func awaitRetirementRegression(t *testing.T, owner *ComposerTarget, id string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := owner.AwaitSubmissionRetirement(ctx, id)
	if err == context.DeadlineExceeded {
		t.Fatal("terminal occurrence left its attachment flight waiting forever")
	}
	return err
}

func applyRetirementEvent(s *Store, typ string, seq float64, data any) {
	raw, _ := json.Marshal(map[string]any{"type": typ, "seq": seq, "data": data})
	s.applyPersistenceEvent(raw)
}

func insertSteeringOccurrence(s *Store, id string) {
	applyRetirementEvent(s, "agent/inbox/spliced", 10, map[string]any{"target": "next-step", "start": 0, "removedCount": 0, "inserted": []any{map[string]any{"source": map[string]string{"kind": "user", "rpcId": id}}}})
}

func TestComposerRetirementAcceptedSteerNeedsObservedHandoff(t *testing.T) {
	for _, terminal := range []string{"canceled", "consumed-unadmitted"} {
		t.Run(terminal, func(t *testing.T) {
			s := newRetirementRegressionStore()
			owner, id := beginAcceptedRetirement(t, s, "a")
			assertRetirementPending(t, owner, id)
			insertSteeringOccurrence(s, id)
			outcome := "canceled"
			if terminal == "consumed-unadmitted" {
				outcome = "consumed"
			}
			applyRetirementEvent(s, "agent/inbox/spliced", 11, map[string]any{"target": "next-step", "start": 0, "removedCount": 1, "inserted": []any{}, "outcome": outcome})
			if terminal == "consumed-unadmitted" {
				assertRetirementPending(t, owner, id) // Consumption alone is not failure.
				applyRetirementEvent(s, "request/context", 12, map[string]any{})
			}
			if err := awaitRetirementRegression(t, owner, id); err == nil {
				t.Fatal("unadmitted steering payload was treated as delivered")
			}
			if len(s.composerPending) != 0 {
				t.Fatal("failed echo remained in the transcript")
			}
		})
	}
}

func TestComposerRetirementObservedMessageCannotBecomeResend(t *testing.T) {
	s := newRetirementRegressionStore()
	owner, id := beginAcceptedRetirement(t, s, "a")
	insertSteeringOccurrence(s, id)
	applyRetirementEvent(s, "user/message", 11, map[string]any{"source": map[string]string{"kind": "user", "rpcId": id}, "content": []any{map[string]string{"type": "text", "text": "retained a"}}})
	applyRetirementEvent(s, "agent/inbox/spliced", 12, map[string]any{"target": "next-step", "start": 0, "removedCount": 1, "inserted": []any{}, "outcome": "canceled"})
	applyRetirementEvent(s, "turn/end", 13, map[string]any{})
	s.applyDeleteResultLocked(api.SessionDeleteValue{DeletedSessionIds: []string{"a"}})
	if err := awaitRetirementRegression(t, owner, id); err != nil {
		t.Fatalf("already observed message was restored as unsent: %v", err)
	}
}

func TestComposerRetirementDurableDeletionIsOwnerScoped(t *testing.T) {
	for _, source := range []string{"live-event", "delete-receipt"} {
		t.Run(source, func(t *testing.T) {
			s := newRetirementRegressionStore()
			a, aid := beginAcceptedRetirement(t, s, "a")
			b, bid := beginAcceptedRetirement(t, s, "b")
			// Replay repairs roster state only, and may precede a new occurrence.
			s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/deleted", sessionID: "a"}, false)
			assertRetirementPending(t, a, aid)
			// Ordinary live removal retains reloadable Session state.
			s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/removed", sessionID: "a"}, true)
			assertRetirementPending(t, a, aid)
			if source == "live-event" {
				s.applyLifecycleMutationLocked(lifecycleMutation{kind: "api-session/deleted", sessionID: "a"}, true)
			} else {
				s.applyDeleteResultLocked(api.SessionDeleteValue{DeletedSessionIds: []string{"a"}})
			}
			if err := awaitRetirementRegression(t, a, aid); err == nil || !strings.Contains(err.Error(), "unconfirmed") {
				t.Fatalf("deleted Session did not retain an explicit uncertain outcome: %v", err)
			}
			assertRetirementPending(t, b, bid)
			s.Detach()
			if err := awaitRetirementRegression(t, b, bid); err == nil {
				t.Fatal("remaining Host-owned flight was silently discarded")
			}
		})
	}
}
