package settings

import (
	"context"
	"encoding/json"
	"errors"
)

const VisualRepliesNamespace = "visual-replies"

// SetVisualReplyEnableCheck installs the desktop's runtime acceptance boundary.
// Called on explicit settings writes, not merely when viewing a namespace.
// The origin is captured with the client so remote settings cannot accidentally
// use a readiness result belonging to the local desktop Host.
func (s *Store) SetVisualReplyEnableCheck(check func(context.Context, string) error) {
	s.mu.Lock()
	s.visualReplyEnableCheck = check
	s.mu.Unlock()
}

func (s *Store) verifyVisualReplyEnable(ctx context.Context, name string, enabled bool, origin string, generation uint64) error {
	if name != VisualRepliesNamespace || !enabled {
		return nil
	}
	s.mu.Lock()
	check := s.visualReplyEnableCheck
	s.mu.Unlock()
	if check == nil {
		return errors.New("HTML visual replies unavailable: desktop runtime verification is not configured")
	}
	if err := check(ctx, origin); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.generation != generation || s.client == nil || s.client.Endpoint.Origin != origin {
		return errors.New("HTML visual replies settings connection changed during verification")
	}
	return ctx.Err()
}
func visualReplyEnabled(section map[string]json.RawMessage) bool {
	var enabled bool
	_ = json.Unmarshal(section["enabled"], &enabled)
	return enabled
}
func visualReplyMutationEnables(ops []json.RawMessage) bool {
	// Only an explicit set enabled=false is provably a disabling mutation.
	for _, raw := range ops {
		var op struct {
			Op    string          `json:"op"`
			Path  []string        `json:"path"`
			Value json.RawMessage `json:"value"`
		}
		if json.Unmarshal(raw, &op) != nil {
			return true
		}
		var value bool
		if op.Op != "set" || len(op.Path) != 1 || op.Path[0] != "enabled" || json.Unmarshal(op.Value, &value) != nil || value {
			return true
		}
	}
	return false
}
