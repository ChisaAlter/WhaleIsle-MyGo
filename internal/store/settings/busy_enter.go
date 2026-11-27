package settings

import (
	"encoding/json"
	"sync"

	"whaleisle/internal/dsh"
)

var (
	busyEnterCacheMu sync.Mutex
	busyEnterCache   = map[string]string{}
)

func parseBusyEnter(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "queue"
	}
	busyEnterCacheMu.Lock()
	if cached, ok := busyEnterCache[string(raw)]; ok {
		busyEnterCacheMu.Unlock()
		return cached
	}
	busyEnterCacheMu.Unlock()

	result := "queue"
	var value struct {
		BusyEnter string `json:"busyEnter"`
	}
	if json.Unmarshal(raw, &value) == nil && value.BusyEnter == "steer" {
		result = "steer"
	}

	busyEnterCacheMu.Lock()
	if len(busyEnterCache) >= 64 {
		clear(busyEnterCache)
	}
	busyEnterCache[string(raw)] = result
	busyEnterCacheMu.Unlock()
	return result
}

// BusyEnterFor reads only the resolved ui-conversation value from the exact
// attached Host. Unsubmitted editor drafts and another Host's cache cannot
// change delivery. Queue is the original plugin's documented default.
func (s *Store) BusyEnterFor(client *dsh.Client) string {
	if s == nil || client == nil {
		return "queue"
	}
	s.mu.Lock()
	defer s.unlockPublish()
	if s.client != client || s.describedClient != client {
		return "queue"
	}
	ns := s.findNamespaceLocked("ui-conversation")
	if ns == nil {
		return "queue"
	}
	return parseBusyEnter(ns.Value)
}
