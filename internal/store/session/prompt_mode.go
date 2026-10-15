package session

import "strings"

// PromptMode is the session/prompt wire mode (queue vs steer / 插话).
type PromptMode string

const (
	PromptModeQueue PromptMode = "queue"
	PromptModeSteer PromptMode = "steer"
)

// NormalizePromptMode maps UI or legacy values to a supported wire mode.
func NormalizePromptMode(mode string) PromptMode {
	switch strings.TrimSpace(mode) {
	case string(PromptModeSteer), "interrupt":
		return PromptModeSteer
	default:
		return PromptModeQueue
	}
}

// SetPromptMode persists the composer send mode in store ViewState.
func (s *Store) SetPromptMode(mode PromptMode) {
	if s == nil {
		return
	}
	mode = NormalizePromptMode(string(mode))
	s.mu.Lock()
	if s.promptMode == mode && s.promptModeExplicit {
		s.mu.Unlock()
		return
	}
	s.promptMode = mode
	s.promptModeExplicit = true
	s.mu.Unlock()
	s.fire()
}
