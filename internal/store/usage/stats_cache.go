package usage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// The managed harness cache belongs to its data directory, not its ephemeral
// listening port. Authentication and account/wallet payloads are never cached.
type overviewCache struct {
	Version int             `json:"version"`
	SavedAt int64           `json:"savedAt"`
	Payload json.RawMessage `json:"payload"`
}

func (s *Store) SetStatsCachePath(path string) {
	s.mu.Lock()
	s.statsCachePath = path
	s.mu.Unlock()
}

func (s *Store) restoreStatsCacheLocked() {
	if s.statsCachePath == "" || s.client == nil {
		return
	}
	data, err := os.ReadFile(s.statsCachePath)
	if err != nil {
		return
	}
	var record overviewCache
	if json.Unmarshal(data, &record) != nil || record.Version != 4 || decodeOverview(record.Payload) != nil {
		return
	}
	s.stats.Raw = append(json.RawMessage(nil), record.Payload...)
	s.stats.Cached = true
}

// Called from the overview worker after its existing generation/endpoint check.
// A failed disk write leaves the successful in-memory overview available.
func (s *Store) saveStatsCacheLocked() {
	if s.statsCachePath == "" {
		return
	}
	data, err := json.Marshal(overviewCache{Version: 4, SavedAt: time.Now().UnixMilli(), Payload: s.stats.Raw})
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(s.statsCachePath), 0700) != nil {
		return
	}
	temp, err := os.CreateTemp(filepath.Dir(s.statsCachePath), "usage-overview-*.tmp")
	if err != nil {
		return
	}
	name := temp.Name()
	defer os.Remove(name)
	_, err = temp.Write(data)
	closeErr := temp.Close()
	if err == nil && closeErr == nil {
		_ = os.Rename(name, s.statsCachePath)
	}
}
