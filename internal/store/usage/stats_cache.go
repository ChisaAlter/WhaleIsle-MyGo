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

func readStatsCacheFile(path string) json.RawMessage {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var record overviewCache
	if json.Unmarshal(data, &record) != nil || record.Version != 4 || decodeOverview(record.Payload) != nil {
		return nil
	}
	return append(json.RawMessage(nil), record.Payload...)
}

// Called from the overview worker outside s.mu so disk I/O never blocks UI Snapshot() reads.
// A failed disk write leaves the successful in-memory overview available.
func writeStatsCacheFile(path string, raw json.RawMessage) {
	if path == "" || len(raw) == 0 {
		return
	}
	data, err := json.Marshal(overviewCache{Version: 4, SavedAt: time.Now().UnixMilli(), Payload: raw})
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "usage-overview-*.tmp")
	if err != nil {
		return
	}
	name := temp.Name()
	defer os.Remove(name)
	_, err = temp.Write(data)
	closeErr := temp.Close()
	if err == nil && closeErr == nil {
		_ = os.Rename(name, path)
	}
}
