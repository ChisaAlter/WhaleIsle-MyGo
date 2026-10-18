package session

import (
	"context"
	"whaleisle/internal/dsh"
)

// NextPromptSuggestion is transient and belongs to one visible composer.
type NextPromptSuggestion struct {
	SessionID string
	Turn, Seq float64
	Text      string
}

type nextPromptRequest struct {
	client       *dsh.Client
	sessionID    string
	completedSeq float64
	cancel       context.CancelFunc
	result       NextPromptSuggestion
}

type nextPromptState struct {
	request  *nextPromptRequest
	accepted map[string]float64
}

func (s *Store) clearNextPromptLocked() {
	if s.nextPrompt.request != nil {
		s.nextPrompt.request.cancel()
		s.nextPrompt.request = nil
	}
}

// SyncNextPrompt tracks the same empty, idle composer lifetime as InputBar.
// An accepted suggestion stays consumed for its completed turn.
func (s *Store) SyncNextPrompt(ctx context.Context, eligible bool) NextPromptSuggestion {
	s.mu.Lock()
	client, id := s.client, s.selectedID
	if !eligible || client == nil || id == "" {
		s.clearNextPromptLocked()
		s.mu.Unlock()
		return NextPromptSuggestion{}
	}
	completedSeq := float64(0)
	for _, message := range s.messages {
		if message.EventType == "turn/end" && message.Seq > completedSeq {
			completedSeq = message.Seq
		}
	}
	current := s.nextPrompt.request
	if current != nil && current.client == client && current.sessionID == id && current.completedSeq == completedSeq {
		result := current.result
		s.mu.Unlock()
		return result
	}
	s.clearNextPromptLocked()
	requestCtx, cancel := context.WithCancel(ctx)
	request := &nextPromptRequest{client: client, sessionID: id, completedSeq: completedSeq, cancel: cancel}
	s.nextPrompt.request = request
	s.mu.Unlock()
	go func() {
		result, err := client.NextPrompt(requestCtx, id)
		s.mu.Lock()
		if s.nextPrompt.request != request || s.client != client || s.selectedID != id {
			s.mu.Unlock()
			return
		}
		if err == nil && result.Text != nil && result.Seq != nil && result.Turn != nil {
			accepted, seen := s.nextPrompt.accepted[id]
			if !seen || accepted != *result.Seq {
				request.result = NextPromptSuggestion{SessionID: id, Turn: *result.Turn, Seq: *result.Seq, Text: *result.Text}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
	return NextPromptSuggestion{}
}

// AcceptNextPrompt consumes the displayed suggestion without submitting a turn.
func (s *Store) AcceptNextPrompt(suggestion NextPromptSuggestion) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	request := s.nextPrompt.request
	if request == nil || request.client != s.client || request.sessionID != s.selectedID || request.result != suggestion {
		return ""
	}
	if s.nextPrompt.accepted == nil {
		s.nextPrompt.accepted = map[string]float64{}
	}
	s.nextPrompt.accepted[s.selectedID] = suggestion.Seq
	request.result = NextPromptSuggestion{}
	return suggestion.Text
}
