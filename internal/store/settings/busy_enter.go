package settings

import (
	"encoding/json"
	"whaleisle/internal/dsh"
)

// BusyEnterFor reads only the resolved ui-conversation value from the exact
// attached Host. Unsubmitted editor drafts and another Host's cache cannot
// change delivery. Queue is the original plugin's documented default.
func (s *Store) BusyEnterFor(client *dsh.Client) string {
	if s == nil || client == nil {
		return "queue"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != client || s.describedClient != client {
		return "queue"
	}
	ns := s.findNamespaceLocked("ui-conversation")
	if ns == nil {
		return "queue"
	}
	var value struct {
		BusyEnter string `json:"busyEnter"`
	}
	if json.Unmarshal(ns.Value, &value) == nil && value.BusyEnter == "steer" {
		return "steer"
	}
	return "queue"
}
