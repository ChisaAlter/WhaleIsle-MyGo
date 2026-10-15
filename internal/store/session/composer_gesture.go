package session

import (
	"errors"
	"whaleisle/internal/dsh"
)

// ResolveBusyEnterMode mirrors the original conversation submission policy.
// An idle/non-steerable address queues; the accelerated chord reverses the
// busy preference. This is an explicit gesture, never a settings side effect.
func ResolveBusyEnterMode(preferred string, running, accelerated, steeringAvailable bool) PromptMode {
	if !running || !steeringAvailable {
		return PromptModeQueue
	}
	mode := PromptModeQueue
	if preferred == "steer" {
		mode = PromptModeSteer
	}
	if accelerated {
		if mode == PromptModeQueue {
			return PromptModeSteer
		}
		return PromptModeQueue
	}
	return mode
}

// CaptureComposerGestureTarget atomically pins the addressed session, Host
// epoch and delivery mode before uploads/admission detach from the editor.
// preference must be a read-only Host-scoped accessor; it must not call Store.
func (s *Store) CaptureComposerGestureTarget(sourceID, targetID string, hostGeneration uint64, accelerated bool, preference func(*dsh.Client) string) (*ComposerTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return nil, ErrNoClient
	}
	if s.composerHostGeneration != hostGeneration {
		return nil, errors.New("composer Host changed; draft retained")
	}
	if s.selectedID != sourceID {
		return nil, errors.New("composer session changed; draft retained")
	}
	preferred := "queue"
	if preference != nil {
		preferred = preference(s.client)
	}
	// Preserve an explicit click on the existing native Queue/Steer controls.
	// UseBusyEnterSetting returns to the persisted Host preference.
	if s.promptModeExplicit {
		preferred = string(s.promptMode)
	}
	running := targetID != "" && targetID == s.selectedID && s.turnStartMs > 0
	// Retained roster rows from a disconnected Host are not running evidence.
	if s.composerRosterClient == s.client {
		for _, item := range s.items {
			if item.SessionID == targetID && item.Running {
				running = true
			}
		}
	}
	// This composer addresses the normal session/prompt transport, which
	// supports steer. Delegated one-shot subagent input uses a separate path.
	mode := ResolveBusyEnterMode(preferred, running, accelerated, targetID != "")
	return &ComposerTarget{SessionID: targetID, store: s, client: s.client, epoch: s.composerHostGeneration, followGeneration: s.followGeneration, mode: mode}, nil
}

// UseBusyEnterSetting removes only the Host-local manual override. It neither
// persists settings nor changes any already captured ComposerTarget.
func (s *Store) UseBusyEnterSetting() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.promptModeExplicit = false
	s.mu.Unlock()
	s.fire()
}
