package interaction

import (
	"context"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// AttachClient binds an optional dsh client for userQuestions/* RPC (waterfall remains primary).
func (s *Store) AttachClient(client *dsh.Client, mux *dsh.Mux) {
	s.mu.Lock()
	s.client = client
	s.mux = mux
	s.mu.Unlock()
}

// AnswerUserQuestionsRPC best-effort calls userQuestions/answer alongside waterfall resolve.
func (s *Store) AnswerUserQuestionsRPC(ctx context.Context, agentID, callID string, answer api.AskUserQuestionAnswer) {
	agentID = strings.TrimSpace(agentID)
	callID = strings.TrimSpace(callID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || callID == "" {
		return
	}
	go func() {
		var out api.AnswerResult
		_ = client.Call(ctx, "userQuestions/answer", map[string]any{
			"agentId": agentID,
			"callId":  callID,
			"answer":  answer,
		}, &out)
	}()
}

// AttachUserQuestionsWait opens userQuestions/attachWait briefly for remainingMs probe.
func (s *Store) AttachUserQuestionsWait(ctx context.Context, mux *dsh.Mux, agentID, callID string) {
	agentID = strings.TrimSpace(agentID)
	callID = strings.TrimSpace(callID)
	if mux == nil || callID == "" {
		return
	}
	go func() {
		if err := mux.Dial(ctx); err != nil {
			return
		}
		stream, err := mux.Open(ctx, "userQuestions/attachWait", map[string]any{
			"args": map[string]any{
				"agentId": agentID,
				"callId":  callID,
			},
		})
		if err != nil {
			return
		}
		defer stream.Cancel()
		_, _ = stream.Recv(ctx)
	}()
}
