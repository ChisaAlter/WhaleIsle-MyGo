package interaction

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

type submissionFixtureScope struct {
	client *dsh.Client
	ctx    context.Context
	active atomic.Bool
	agent  string
}

func (s *submissionFixtureScope) Client() *dsh.Client      { return s.client }
func (s *submissionFixtureScope) Context() context.Context { return s.ctx }
func (s *submissionFixtureScope) AgentID() string          { return s.agent }
func (s *submissionFixtureScope) Valid() bool              { return s.active.Load() }
func scopedFixture(s *Store) (*ScopedStore, *submissionFixtureScope) {
	scope := &submissionFixtureScope{client: &dsh.Client{}, ctx: context.Background(), agent: "A"}
	scope.active.Store(true)
	s.AttachClient(scope.client, nil)
	return NewScopedStore(s, scope), scope
}
func TestScopedInteractionApprovalDeliveryAndRetry(t *testing.T) {
	s := New()
	s.pending = []Pending{{AgentID: "A", EventID: "e", Event: EventApproval}}
	s.waiters["e"] = make(chan dsh.EventResultOutcome, 1)
	scoped, scope := scopedFixture(s)
	if len(s.waiters["e"]) != 0 {
		t.Fatal("facade creation sent")
	}
	scoped.ResolveAllow("e")
	out := <-s.waiters["e"]
	if out.Delivery == nil || !out.Delivery.Valid() {
		t.Fatal("approval missing delivery scope")
	}
	s.OnResult(dsh.EventResult{EventID: "e", Outcome: out}, errors.New("retry"))
	scope.active.Store(false)
	scoped.Retry("e")
	if len(s.waiters["e"]) != 0 || out.Delivery.Valid() {
		t.Fatal("revocation allowed retry/delivery")
	}
	scope.active.Store(true)
	scoped.Retry("e")
	retry := <-s.waiters["e"]
	if string(out.Value) != string(retry.Value) || retry.Delivery == out.Delivery {
		t.Fatal("retry did not renew exact decision proof")
	}
	if s.OnResult(dsh.EventResult{EventID: "e", Outcome: out}, nil) || len(s.Snapshot().Pending) != 1 {
		t.Fatal("old delivery completed newer retry")
	}
	s.OnResult(dsh.EventResult{EventID: "e", Outcome: retry}, nil)
	if len(s.Snapshot().Pending) != 0 {
		t.Fatal("current delivery did not settle")
	}
}
func TestScopedInteractionAllSubmissionGesturesFailClosed(t *testing.T) {
	actions := []struct {
		name string
		run  func(*ScopedStore)
	}{
		{"allow", func(s *ScopedStore) { s.ResolveAllow("e") }}, {"deny", func(s *ScopedStore) { s.ResolveDeny("e") }},
		{"questions", func(s *ScopedStore) { s.ResolveQuestions("e") }}, {"plan", func(s *ScopedStore) { s.ApprovePlanReview("e") }},
		{"dismiss", func(s *ScopedStore) { s.DismissPlanReview("e") }}, {"retry", func(s *ScopedStore) { s.Retry("e") }},
		{"enter", func(s *ScopedStore) { s.ContinueQuestion("A", "e", "last") }}, {"option enter", func(s *ScopedStore) { s.SubmitQuestionBatch("A", "e") }},
		{"skip", func(s *ScopedStore) { s.SkipQuestionAndContinue("A", "e", "last") }},
	}
	for _, a := range actions {
		t.Run(a.name, func(t *testing.T) {
			s := questionWizardFixture()
			scoped, scope := scopedFixture(s)
			s.SetQuestionCustom("e", "first", "one")
			s.SetQuestionCustom("e", "last", "two")
			s.NavigateQuestion("A", "e", 1)
			scope.active.Store(false)
			a.run(scoped)
			if len(s.waiters["e"]) != 0 || s.Snapshot().Pending[0].Sending {
				t.Fatal("stale callback bypassed scope")
			}
			scope.active.Store(true)
			scope.agent = "B"
			a.run(scoped)
			if len(s.waiters["e"]) != 0 {
				t.Fatal("foreign child answered")
			}
			scope.agent = "A"
			s.AttachClient(&dsh.Client{}, nil)
			a.run(scoped)
			if len(s.waiters["e"]) != 0 {
				t.Fatal("foreign Host answered")
			}
		})
	}
}
func TestScopedInteractionQuestionAndPlanUseGuardedSender(t *testing.T) {
	for _, gesture := range []string{"enter", "batch", "skip", "plan", "dismiss"} {
		t.Run(gesture, func(t *testing.T) {
			s := questionWizardFixture()
			scoped, _ := scopedFixture(s)
			s.SetQuestionCustom("e", "first", "one")
			s.SetQuestionCustom("e", "last", "two")
			s.NavigateQuestion("A", "e", 1)
			switch gesture {
			case "enter":
				scoped.ContinueQuestion("A", "e", "last")
			case "batch":
				scoped.SubmitQuestionBatch("A", "e")
			case "skip":
				scoped.SkipQuestionAndContinue("A", "e", "last")
			default:
				s.pending[0].CallID = ""
				s.questionDrafts["e"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{{ID: "plan", Detail: "# Plan", Intent: &planReviewIntent{Kind: "plan-review", Approve: "Approve"}, Options: []userQuestionOption{{Label: "Approve"}}}}})
				if gesture == "plan" {
					scoped.ApprovePlanReview("e")
				} else {
					scoped.DismissPlanReview("e")
				}
			}
			select {
			case out := <-s.waiters["e"]:
				if out.Delivery == nil || !out.Delivery.Valid() {
					t.Fatal("unguarded send")
				}
			case <-time.After(time.Second):
				t.Fatal("gesture failed to send")
			}
		})
	}
}

type scopedRoundTrip func(*http.Request) (*http.Response, error)

func (f scopedRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestScopedInteractionLateAnswerRevocationCancelsRPC(t *testing.T) {
	s := New()
	scoped, scope := scopedFixture(s)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scope.ctx = ctx
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	scope.client.Endpoint.Origin = "http://fixture.invalid"
	scope.client.HTTP = &http.Client{Transport: scopedRoundTrip(func(r *http.Request) (*http.Response, error) {
		var body struct {
			Payload struct {
				Args struct{ AgentID, CallID string } `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Payload.Args.AgentID != "A" || body.Payload.Args.CallID != "call-1" {
			t.Error("RPC borrowed another parent")
		}
		close(entered)
		<-r.Context().Done()
		close(cancelled)
		return nil, r.Context().Err()
	})}
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture)
	id := s.SnapshotForAgent("A").Pending[0].EventID
	s.SetQuestionSingleSelect(id, "q", "B")
	scoped.ResolveQuestions(id)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("late RPC did not start")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("late RPC not cancelled")
	}
	awaitQuestionFixture(t, func() bool {
		p := s.SnapshotForAgent("A").Pending
		return len(p) == 1 && !p[0].Sending && p[0].Error != ""
	})
	if s.SnapshotForAgent("A").QuestionStatus == "submitted" {
		t.Fatal("cancelled reply reported sent")
	}
}
