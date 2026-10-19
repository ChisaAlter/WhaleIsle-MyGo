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
	generation   uint64
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
	if !eligible || ctx.Err() != nil || client == nil || !s.nextPromptSessionEligibleLocked() {
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
	if current != nil && current.client == client && current.sessionID == id && current.generation == s.followGeneration && current.completedSeq == completedSeq {
		result := current.result
		s.mu.Unlock()
		return result
	}
	s.clearNextPromptLocked()
	requestCtx, cancel := context.WithCancel(ctx)
	request := &nextPromptRequest{client: client, sessionID: id, generation: s.followGeneration, completedSeq: completedSeq, cancel: cancel}
	s.nextPrompt.request = request
	s.mu.Unlock()
	go func() {
		result, err := client.NextPrompt(requestCtx, id)
		s.mu.Lock()
		if requestCtx.Err() != nil || s.nextPrompt.request != request || s.client != client || s.selectedID != id || s.followGeneration != request.generation || !s.nextPromptSessionEligibleLocked() {
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
	if request == nil || request.client != s.client || request.sessionID != s.selectedID || request.generation != s.followGeneration || request.result != suggestion || !s.nextPromptSessionEligibleLocked() {
		return ""
	}
	if s.nextPrompt.accepted == nil {
		s.nextPrompt.accepted = map[string]float64{}
	}
	s.nextPrompt.accepted[s.selectedID] = suggestion.Seq
	request.result = NextPromptSuggestion{}
	return suggestion.Text
}

// The native frame owns draft/popup and interaction gates; the Store checks the
// current Session again at asynchronous completion and acceptance, like the
// original aborted effect and keyboard.snapshot checks. A retained suggestion
// must not cross a reopen of the same ID or a managed-presentation transition.
func (s *Store) nextPromptSessionEligibleLocked() bool {
	if s.selectedID == "" || !s.historyReady || s.historyLoading || s.historyError != "" || s.promptBusy || s.composerPromptBusyLocked() || s.composerPendingForSelectedLocked() || s.historyBranchBusy || s.turnStartMs > 0 {
		return false
	}
	for _, item := range s.items {
		if item.SessionID == s.selectedID {
			return !item.Running && !item.Blank && item.Origin != "subagent" && (item.Presentation == nil || item.Presentation.Composer != "managed")
		}
	}
	return false
}
