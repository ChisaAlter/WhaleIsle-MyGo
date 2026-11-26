package interaction

import (
	"context"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type testScope struct {
	client  *dsh.Client
	ctx     context.Context
	agentID string
}

func (t testScope) Client() *dsh.Client      { return t.client }
func (t testScope) Context() context.Context { return t.ctx }
func (t testScope) AgentID() string          { return t.agentID }
func (t testScope) Valid() bool              { return true }

func TestScopedStoreAndQuestionWaitCloseDupDoneNonBlocking(t *testing.T) {
	t.Run("scoped store closes dupDone without blocking on full waiter", func(t *testing.T) {
		client := &dsh.Client{}
		s := New()
		s.AttachClient(client, nil)
		id := "ev-scoped-1"
		waiter := make(chan dsh.EventResultOutcome, 1)
		// Pre-fill waiter so a blocking send under s.mu would deadlock.
		waiter <- dsh.EventResultOutcome{Kind: "resolved"}
		dup := make(chan struct{})
		s.mu.Lock()
		s.pending = append(s.pending, Pending{EventID: id, AgentID: "child-1", Event: EventApproval})
		s.waiters[id] = waiter
		if s.dupDones == nil {
			s.dupDones = make(map[string]chan struct{})
		}
		s.dupDones[id] = dup
		s.mu.Unlock()

		scoped := NewScopedStore(s, testScope{client: client, ctx: context.Background(), agentID: "child-1"})
		done := make(chan struct{})
		go func() {
			scoped.resolveScoped(id, approvalOutcome(outcomeAllowedOnce))
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("resolveScoped blocked while holding store mutex on full waiter")
		}
		select {
		case <-dup:
		default:
			t.Fatal("resolveScoped did not close dupDone")
		}
	})

	t.Run("question wait timeout closes dupDone without blocking on full waiter", func(t *testing.T) {
		s := New()
		id := "ev-q-timeout"
		waiter := make(chan dsh.EventResultOutcome, 1)
		waiter <- dsh.EventResultOutcome{Kind: "resolved"}
		dup := make(chan struct{})
		s.mu.Lock()
		s.pending = append(s.pending, Pending{EventID: id, AgentID: "a1", CallID: "c1", Event: EventUserQuestions, questionProjected: true})
		s.waiters[id] = waiter
		if s.questionDrafts == nil {
			s.questionDrafts = make(map[string]*draftSession)
		}
		s.questionDrafts[id] = &draftSession{}
		if s.dupDones == nil {
			s.dupDones = make(map[string]chan struct{})
		}
		s.dupDones[id] = dup
		s.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done := make(chan struct{})
		go func() {
			s.runQuestionWait(ctx, "a1", "c1", id, 10*time.Millisecond)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("runQuestionWait blocked on full waiter at timeout")
		}
		select {
		case <-dup:
		default:
			t.Fatal("runQuestionWait timeout did not close dupDone")
		}
	})
}
