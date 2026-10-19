package settings

import (
	"context"
	"encoding/json"
	"sort"
	"whaleisle/internal/dsh"
)

// Mirrors pinned desktop welcome-backend.read: actual provider addresses name
// credential references, then describe returns only configured/writable facts.
func (s *Store) readOnboardingAPIKeyPresence(client *dsh.Client, generation uint64) bool {
	s.mu.Lock()
	if s.client != client || s.generation != generation {
		s.mu.Unlock()
		return false
	}
	values := map[string]json.RawMessage{}
	for _, ns := range s.namespaces {
		values[ns.NS] = append(json.RawMessage(nil), ns.Value...)
	}
	s.mu.Unlock()
	refs := map[string]bool{}
	add := func(raw json.RawMessage, path []string) {
		var value any
		if json.Unmarshal(raw, &value) != nil {
			return
		}
		for _, key := range path {
			object, _ := value.(map[string]any)
			value = object[key]
		}
		object, _ := value.(map[string]any)
		if ref, ok := object["apiKeyEnv"].(string); ok {
			refs[ref] = true
		}
	}
	add(values["llm-deepseek"], nil)
	var providers []struct {
		Namespace string   `json:"settingsNs"`
		Path      []string `json:"settingsPath"`
	}
	if client.Call(context.Background(), "llm/listConfigurableProviders", nil, &providers) != nil {
		return false
	}
	for _, p := range providers {
		add(values[p.Namespace], p.Path)
	}
	keys := make([]string, 0, len(refs))
	for key := range refs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for start := 0; start < len(keys); start += 64 {
		var states map[string]struct {
			Configured bool `json:"configured"`
		}
		if client.Call(context.Background(), "credentials/describe", map[string]any{"refs": keys[start:min(start+64, len(keys))]}, &states) != nil {
			return false
		}
		for _, state := range states {
			if state.Configured {
				return true
			}
		}
	}
	return false
}
