package interaction

import (
	"context"
	"encoding/json"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// AttachClient binds an optional dsh client for userQuestions/* RPC (waterfall remains primary).
func (s *Store) AttachClient(client *dsh.Client, mux *dsh.Mux) {
	s.mu.Lock()
	if s.client != client {
		s.resetQuestionRecoveryLocked()
	}
	s.client = client
	s.mux = mux
	s.mu.Unlock()
}

// AnswerUserQuestionsRPC submits a recovered continued question. Live requests
// retain their waterfall outcome contract and never send a duplicate RPC.
func (s *Store) AnswerUserQuestionsRPC(ctx context.Context, agentID, callID string, answer api.AskUserQuestionAnswer) {
	s.submitQuestionAnswer(ctx, agentID, callID, answer)
}

// attachQuestionWait holds the timed Host claim until answer, cancellation, or timeout.
func (s *Store) attachQuestionWait(ctx context.Context, mux *dsh.Mux, agentID, callID, eventID string) {
	if mux == nil || callID == "" {
		return
	}
	go func() {
		if err := mux.Dial(ctx); err != nil {
			return
		}
		stream, err := mux.Open(ctx, "userQuestions/attachWait", map[string]any{"args": map[string]any{"agentId": agentID, "callId": callID}})
		if err != nil {
			return
		}
		defer stream.Cancel()
		raw, err := stream.Recv(ctx)
		if err != nil {
			return
		}
		var wait api.AttachWaitResult
		if json.Unmarshal(raw, &wait) != nil || wait.RemainingMs < 0 || wait.RemainingMs > 2147483647 {
			return
		}
		timer := time.NewTimer(time.Duration(wait.RemainingMs) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
			s.resolve(eventID, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "UserQuestionError", Code: "ASK_TIMED_OUT", Message: "Question timed out"}})
		}
	}()
}
